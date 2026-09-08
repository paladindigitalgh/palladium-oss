package kontron_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron"
)

// realBlacklistSample is the exact, real output captured from a
// production Kontron/Iskratel C16 running "show onu black-list" — the
// one confirmed sample kontron.ParseBlacklistEntries is built from.
const realBlacklistSample = "jamestown-allen-olt-02#show onu black-list\n" +
	"Interface  Serial Number     Password/Registration Id                Cause\n" +
	"---------  ----------------  --------------------------------------  -----------------------\n" +
	"xgs/6      ISKT2308DD88      \"\"                                      Serial Number not known\n" +
	"---------------------------\n" +
	"Total: 1\n"

func TestParseBlacklistEntriesRealSample(t *testing.T) {
	entries, err := kontron.ParseBlacklistEntries(realBlacklistSample)
	if err != nil {
		t.Fatalf("ParseBlacklistEntries() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1", len(entries))
	}

	got := entries[0]
	want := kontron.BlacklistEntry{
		Interface:      "xgs/6",
		SerialNumber:   "ISKT2308DD88",
		RegistrationID: `""`,
		Cause:          "Serial Number not known",
	}
	if got != want {
		t.Errorf("entries[0] = %+v, want %+v", got, want)
	}
}

// TestParseBlacklistEntriesEmptyInput exercises an empty response, which
// is not the confirmed zero-entries shape (see ParseBlacklistEntries'
// own doc comment on why that specific shape is still unverified) — this
// only proves the parser does not error or panic on the total absence of
// any table at all.
func TestParseBlacklistEntriesEmptyInput(t *testing.T) {
	entries, err := kontron.ParseBlacklistEntries("")
	if err != nil {
		t.Fatalf("ParseBlacklistEntries() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

// TestParseBlacklistEntriesNoColumnRuleLine covers a response with a
// banner and prose but no recognizable column rule line at all —
// synthesized input, not a real captured sample, standing in for
// whatever an OLT with nothing blacklisted might print. Per
// ParseBlacklistEntries' own doc comment, this is treated the same as
// zero entries, not an error.
func TestParseBlacklistEntriesNoColumnRuleLine(t *testing.T) {
	entries, err := kontron.ParseBlacklistEntries("jamestown-allen-olt-02#show onu black-list\nTotal: 0\n")
	if err != nil {
		t.Fatalf("ParseBlacklistEntries() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

// TestParseBlacklistEntriesTotalMismatch is synthesized input (not a
// real capture) that deliberately breaks the real sample's Total by one,
// proving the sanity check described in ParseBlacklistEntries' doc
// comment actually fires instead of silently returning a wrong count.
func TestParseBlacklistEntriesTotalMismatch(t *testing.T) {
	broken := "jamestown-allen-olt-02#show onu black-list\n" +
		"Interface  Serial Number     Password/Registration Id                Cause\n" +
		"---------  ----------------  --------------------------------------  -----------------------\n" +
		"xgs/6      ISKT2308DD88      \"\"                                      Serial Number not known\n" +
		"---------------------------\n" +
		"Total: 2\n"

	if _, err := kontron.ParseBlacklistEntries(broken); err == nil {
		t.Fatal("ParseBlacklistEntries() error = nil, want a mismatch error")
	}
}
