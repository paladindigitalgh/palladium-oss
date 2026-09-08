package kontron_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	diagnosticskontron "github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron"
	"github.com/paladindigitalgh/palladium-oss/internal/platform/ssh"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning/kontron"
)

// fakeShell is a minimal ssh.Shell fake, letting these tests script
// exactly what each RunCommand call returns without a real SSH
// connection — the same technique
// internal/diagnostics/kontron/kontron_test.go uses for its own Client
// tests.
type fakeShell struct {
	// outputs maps a command to what RunCommand should return for it.
	// Any command not present returns "" (success) by default.
	outputs map[string]string
	err     error
	calls   []string
}

func (f *fakeShell) RunCommand(_ context.Context, command string, _ ...ssh.PagerPrompt) (string, error) {
	f.calls = append(f.calls, command)
	if f.err != nil {
		return "", f.err
	}
	return f.outputs[command], nil
}

func (f *fakeShell) Close() error { return nil }

var _ ssh.Shell = (*fakeShell)(nil)

func TestAuthorizeONUSucceedsAndRunsAllSevenStepsInOrder(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{}}
	client := kontron.NewClient(shell)

	if err := client.AuthorizeONU(context.Background(), "xgs/6/3", "ISKT2308DD88", "iphost"); err != nil {
		t.Fatalf("AuthorizeONU() = %v", err)
	}

	want := []string{
		"configure",
		"interface xgs/6/3",
		"onu serial-number ISKT2308DD88",
		"service-profile iphost",
		"exit",
		"exit",
		"save config",
	}
	if len(shell.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", shell.calls, want)
	}
	for i, c := range want {
		if shell.calls[i] != c {
			t.Errorf("calls[%d] = %q, want %q", i, shell.calls[i], c)
		}
	}
}

// TestAuthorizeONUAbortsOnFirstNonEmptyOutput proves the generic
// failure-handling AuthorizeONU's own doc comment describes: any
// non-empty output aborts immediately, with no special-casing of known
// failure strings, and no further steps run.
func TestAuthorizeONUAbortsOnFirstNonEmptyOutput(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{
		"onu serial-number ISKT1234": "onu_serial_number wrong size!",
	}}
	client := kontron.NewClient(shell)

	err := client.AuthorizeONU(context.Background(), "xgs/6/3", "ISKT1234", "iphost")
	if err == nil {
		t.Fatal("AuthorizeONU() error = nil, want an error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "onu_serial_number wrong size!") {
		t.Errorf("AuthorizeONU() error = %q, want it to contain the device's raw message", got)
	}

	// Nothing after the failing step must have run.
	want := []string{"configure", "interface xgs/6/3", "onu serial-number ISKT1234"}
	if len(shell.calls) != len(want) {
		t.Fatalf("calls = %v, want exactly %v (save config must not run)", shell.calls, want)
	}
}

func TestAuthorizeONURejectsInterfaceWithNewline(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{}}
	client := kontron.NewClient(shell)

	err := client.AuthorizeONU(context.Background(), "xgs/6/3\nrm -rf /", "ISKT2308DD88", "iphost")
	if !errors.Is(err, kontron.ErrInvalidInterface) {
		t.Errorf("AuthorizeONU() error = %v, want ErrInvalidInterface", err)
	}
	if len(shell.calls) != 0 {
		t.Errorf("calls = %v, want none (validation must happen before anything runs)", shell.calls)
	}
}

func TestAuthorizeONURejectsSerialNumberWithNewline(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{}}
	client := kontron.NewClient(shell)

	err := client.AuthorizeONU(context.Background(), "xgs/6/3", "ISKT2308DD88\nconfigure", "iphost")
	if !errors.Is(err, kontron.ErrInvalidSerialNumber) {
		t.Errorf("AuthorizeONU() error = %v, want ErrInvalidSerialNumber", err)
	}
	if len(shell.calls) != 0 {
		t.Errorf("calls = %v, want none (validation must happen before anything runs)", shell.calls)
	}
}

func TestAuthorizeONURejectsManagementServiceProfileWithNewline(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{}}
	client := kontron.NewClient(shell)

	err := client.AuthorizeONU(context.Background(), "xgs/6/3", "ISKT2308DD88", "iphost\nconfigure")
	if !errors.Is(err, kontron.ErrInvalidManagementServiceProfile) {
		t.Errorf("AuthorizeONU() error = %v, want ErrInvalidManagementServiceProfile", err)
	}
	if len(shell.calls) != 0 {
		t.Errorf("calls = %v, want none (validation must happen before anything runs)", shell.calls)
	}
}

// TestApplyServiceProfileSucceedsWithRealisticDeviceEcho proves the
// actual bug fix confirmed against a real Kontron/Iskratel C16: raw
// output for a genuinely successful command is not empty, it is the
// device's own echo of the command just sent (e.g. "configure" itself
// produced literal output "configure\n\r") — this must still be
// recognized as success, not mistaken for a device-reported failure the
// way it was before stripEcho existed.
func TestApplyServiceProfileSucceedsWithRealisticDeviceEcho(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{
		"configure":                       "configure\n\r",
		"interface xgs/6/3":               "interface xgs/6/3\n\r",
		"service-profile residential-500": "service-profile residential-500\n\r",
		"exit":                            "exit\n\r",
		"save config":                     "save config\n\r",
	}}
	client := kontron.NewClient(shell)

	if err := client.ApplyServiceProfile(context.Background(), "xgs/6/3", "residential-500"); err != nil {
		t.Fatalf("ApplyServiceProfile() = %v, want success despite the device echoing every command back", err)
	}
}

// TestApplyServiceProfileAbortsOnFirstNonEmptyOutputWithRealisticEcho
// proves a genuine device-reported failure is still correctly detected
// once its own echo prefix (present on every response, not just
// failures) is stripped off first.
func TestApplyServiceProfileAbortsOnFirstNonEmptyOutputWithRealisticEcho(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{
		"configure":                       "configure\n\r",
		"interface xgs/6/3":               "interface xgs/6/3\n\r",
		"service-profile residential-500": "service-profile residential-500\r\nunknown service profile\r\n",
	}}
	client := kontron.NewClient(shell)

	err := client.ApplyServiceProfile(context.Background(), "xgs/6/3", "residential-500")
	if err == nil {
		t.Fatal("ApplyServiceProfile() error = nil, want an error")
	}
	if got := err.Error(); !strings.Contains(got, "unknown service profile") || strings.Contains(got, "service-profile residential-500\r\nunknown") {
		t.Errorf("ApplyServiceProfile() error = %q, want the echoed command stripped and only the real device message left", got)
	}
}

func TestApplyServiceProfileSucceedsAndRunsAllSixStepsInOrder(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{}}
	client := kontron.NewClient(shell)

	if err := client.ApplyServiceProfile(context.Background(), "xgs/6/3", "residential-500"); err != nil {
		t.Fatalf("ApplyServiceProfile() = %v", err)
	}

	want := []string{
		"configure",
		"interface xgs/6/3",
		"service-profile residential-500",
		"exit",
		"exit",
		"save config",
	}
	if len(shell.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", shell.calls, want)
	}
	for i, c := range want {
		if shell.calls[i] != c {
			t.Errorf("calls[%d] = %q, want %q", i, shell.calls[i], c)
		}
	}
}

// TestApplyServiceProfileAbortsOnFirstNonEmptyOutput proves
// ApplyServiceProfile follows AuthorizeONU's exact generic
// failure-handling convention: any non-empty output aborts immediately,
// with no further steps run.
func TestApplyServiceProfileAbortsOnFirstNonEmptyOutput(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{
		"service-profile residential-500": "unknown service profile",
	}}
	client := kontron.NewClient(shell)

	err := client.ApplyServiceProfile(context.Background(), "xgs/6/3", "residential-500")
	if err == nil {
		t.Fatal("ApplyServiceProfile() error = nil, want an error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "unknown service profile") {
		t.Errorf("ApplyServiceProfile() error = %q, want it to contain the device's raw message", got)
	}

	want := []string{"configure", "interface xgs/6/3", "service-profile residential-500"}
	if len(shell.calls) != len(want) {
		t.Fatalf("calls = %v, want exactly %v (save config must not run)", shell.calls, want)
	}
}

func TestApplyServiceProfileRejectsInterfaceWithNewline(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{}}
	client := kontron.NewClient(shell)

	err := client.ApplyServiceProfile(context.Background(), "xgs/6/3\nrm -rf /", "residential-500")
	if !errors.Is(err, kontron.ErrInvalidInterface) {
		t.Errorf("ApplyServiceProfile() error = %v, want ErrInvalidInterface", err)
	}
	if len(shell.calls) != 0 {
		t.Errorf("calls = %v, want none (validation must happen before anything runs)", shell.calls)
	}
}

func TestApplyServiceProfileRejectsProfileNameWithNewline(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{}}
	client := kontron.NewClient(shell)

	err := client.ApplyServiceProfile(context.Background(), "xgs/6/3", "residential-500\nconfigure")
	if !errors.Is(err, kontron.ErrInvalidProfileName) {
		t.Errorf("ApplyServiceProfile() error = %v, want ErrInvalidProfileName", err)
	}
	if len(shell.calls) != 0 {
		t.Errorf("calls = %v, want none (validation must happen before anything runs)", shell.calls)
	}
}

func TestRemoveServiceProfileSucceedsAndRunsAllSixStepsInOrder(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{}}
	client := kontron.NewClient(shell)

	if err := client.RemoveServiceProfile(context.Background(), "xgs/6/3", "residential-500"); err != nil {
		t.Fatalf("RemoveServiceProfile() = %v", err)
	}

	want := []string{
		"configure",
		"interface xgs/6/3",
		"no service-profile residential-500",
		"exit",
		"exit",
		"save config",
	}
	if len(shell.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", shell.calls, want)
	}
	for i, c := range want {
		if shell.calls[i] != c {
			t.Errorf("calls[%d] = %q, want %q", i, shell.calls[i], c)
		}
	}
}

func TestRemoveServiceProfileAbortsOnFirstNonEmptyOutput(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{
		"no service-profile residential-500": "profile not assigned to interface",
	}}
	client := kontron.NewClient(shell)

	err := client.RemoveServiceProfile(context.Background(), "xgs/6/3", "residential-500")
	if err == nil {
		t.Fatal("RemoveServiceProfile() error = nil, want an error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "profile not assigned to interface") {
		t.Errorf("RemoveServiceProfile() error = %q, want it to contain the device's raw message", got)
	}

	want := []string{"configure", "interface xgs/6/3", "no service-profile residential-500"}
	if len(shell.calls) != len(want) {
		t.Fatalf("calls = %v, want exactly %v (save config must not run)", shell.calls, want)
	}
}

func TestRemoveServiceProfileRejectsInterfaceWithNewline(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{}}
	client := kontron.NewClient(shell)

	err := client.RemoveServiceProfile(context.Background(), "xgs/6/3\nrm -rf /", "residential-500")
	if !errors.Is(err, kontron.ErrInvalidInterface) {
		t.Errorf("RemoveServiceProfile() error = %v, want ErrInvalidInterface", err)
	}
	if len(shell.calls) != 0 {
		t.Errorf("calls = %v, want none (validation must happen before anything runs)", shell.calls)
	}
}

func TestRemoveServiceProfileRejectsProfileNameWithNewline(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{}}
	client := kontron.NewClient(shell)

	err := client.RemoveServiceProfile(context.Background(), "xgs/6/3", "residential-500\nconfigure")
	if !errors.Is(err, kontron.ErrInvalidProfileName) {
		t.Errorf("RemoveServiceProfile() error = %v, want ErrInvalidProfileName", err)
	}
	if len(shell.calls) != 0 {
		t.Errorf("calls = %v, want none (validation must happen before anything runs)", shell.calls)
	}
}

func TestDeauthorizeONUSucceedsAndRunsAllSixStepsInOrder(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{}}
	client := kontron.NewClient(shell)

	if err := client.DeauthorizeONU(context.Background(), "xgs/6/3"); err != nil {
		t.Fatalf("DeauthorizeONU() = %v", err)
	}

	want := []string{
		"configure",
		"interface xgs/6/3",
		"no onu serial-number",
		"exit",
		"exit",
		"save config",
	}
	if len(shell.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", shell.calls, want)
	}
	for i, c := range want {
		if shell.calls[i] != c {
			t.Errorf("calls[%d] = %q, want %q", i, shell.calls[i], c)
		}
	}
}

func TestDeauthorizeONUAbortsOnFirstNonEmptyOutput(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{
		"no onu serial-number": "no onu configured on this interface",
	}}
	client := kontron.NewClient(shell)

	err := client.DeauthorizeONU(context.Background(), "xgs/6/3")
	if err == nil {
		t.Fatal("DeauthorizeONU() error = nil, want an error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "no onu configured on this interface") {
		t.Errorf("DeauthorizeONU() error = %q, want it to contain the device's raw message", got)
	}

	want := []string{"configure", "interface xgs/6/3", "no onu serial-number"}
	if len(shell.calls) != len(want) {
		t.Fatalf("calls = %v, want exactly %v (save config must not run)", shell.calls, want)
	}
}

func TestDeauthorizeONURejectsInterfaceWithNewline(t *testing.T) {
	shell := &fakeShell{outputs: map[string]string{}}
	client := kontron.NewClient(shell)

	err := client.DeauthorizeONU(context.Background(), "xgs/6/3\nrm -rf /")
	if !errors.Is(err, kontron.ErrInvalidInterface) {
		t.Errorf("DeauthorizeONU() error = %v, want ErrInvalidInterface", err)
	}
	if len(shell.calls) != 0 {
		t.Errorf("calls = %v, want none (validation must happen before anything runs)", shell.calls)
	}
}

func TestNextFreeIndexFillsGaps(t *testing.T) {
	entries := []diagnosticskontron.ONUSummaryEntry{
		{Interface: "xgs/6/1"},
		{Interface: "xgs/6/3"},
	}
	if got := kontron.NextFreeIndex(entries, "xgs/6"); got != 2 {
		t.Errorf("NextFreeIndex() = %d, want 2", got)
	}
}

func TestNextFreeIndexReturnsOneForAnUnusedPort(t *testing.T) {
	entries := []diagnosticskontron.ONUSummaryEntry{
		{Interface: "xgs/1/1"},
	}
	if got := kontron.NextFreeIndex(entries, "xgs/6"); got != 1 {
		t.Errorf("NextFreeIndex() = %d, want 1", got)
	}
}

func TestNextFreeIndexIgnoresOtherPorts(t *testing.T) {
	entries := []diagnosticskontron.ONUSummaryEntry{
		{Interface: "xgs/6/1"},
		{Interface: "xgs/6/2"},
		{Interface: "xgs/7/1"},
	}
	if got := kontron.NextFreeIndex(entries, "xgs/6"); got != 3 {
		t.Errorf("NextFreeIndex() = %d, want 3", got)
	}
}
