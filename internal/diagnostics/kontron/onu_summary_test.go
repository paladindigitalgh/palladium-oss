package kontron_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron"
)

// realONUSummarySample is real output captured from a production
// Kontron/Iskratel C16 running "show onu int all" (truncated from a
// ~20-row original to the rows this test actually asserts against) —
// the one confirmed sample kontron.ParseONUSummaryEntries is built from.
const realONUSummarySample = "jamestown-steele-olt-03#show onu int all\n" +
	"ONU       Oper     Admin    ONU               Service  Serial\n" +
	"Interface State    State    State             State    Number       Password/Registration Id               IP Address      MAC Address       Description\n" +
	"--------- -------- -------- ----------------  -------- ------------ -------------------------------------- --------------- ----------------- ------------------\n" +
	"xgs/1/1   Up       Enable   Active            Enable   ISKT230660D0 \"\"                                     10.70.180.25    48:55:41:06:60:D0 ONU xgs/1/1\n" +
	"xgs/3/1   Down     Enable   Not Configured    Enable   ISKT23066250 \"\"                                     0.0.0.0         00:00:00:00:00:00 ONU xgs/3/1\n" +
	"xgs/6/1   Up       Enable   Active            Enable   ISKT235A81D8 \"\"                                     10.70.180.4     48:55:41:5A:81:D8 ISKT235A81D8\n" +
	"xgs/6/2   Up       Enable   Active            Enable   ISKT2308D850 \"\"                                     10.70.178.208   48:55:41:08:D8:50 ISKT2308D850\n"

func TestParseONUSummaryEntriesRealSample(t *testing.T) {
	entries, err := kontron.ParseONUSummaryEntries(realONUSummarySample)
	if err != nil {
		t.Fatalf("ParseONUSummaryEntries() error = %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("len(entries) = %d, want 4", len(entries))
	}

	// xgs/1/1: a normal, Up/Active row, to prove the common case parses.
	want0 := kontron.ONUSummaryEntry{
		Interface:      "xgs/1/1",
		OperState:      "Up",
		AdminState:     "Enable",
		ONUState:       "Active",
		ServiceState:   "Enable",
		SerialNumber:   "ISKT230660D0",
		RegistrationID: `""`,
		IPAddress:      "10.70.180.25",
		MACAddress:     "48:55:41:06:60:D0",
		Description:    "ONU xgs/1/1",
	}
	if entries[0] != want0 {
		t.Errorf("entries[0] = %+v, want %+v", entries[0], want0)
	}

	// xgs/3/1: Down/"Not Configured" (a multi-word ONU State value) and
	// an all-zero MAC/IP, proving state fields parse independently of
	// the Up/Active rows and that a multi-word column value in the
	// middle of the table doesn't throw off later columns.
	want1 := kontron.ONUSummaryEntry{
		Interface:      "xgs/3/1",
		OperState:      "Down",
		AdminState:     "Enable",
		ONUState:       "Not Configured",
		ServiceState:   "Enable",
		SerialNumber:   "ISKT23066250",
		RegistrationID: `""`,
		IPAddress:      "0.0.0.0",
		MACAddress:     "00:00:00:00:00:00",
		Description:    "ONU xgs/3/1",
	}
	if entries[1] != want1 {
		t.Errorf("entries[1] = %+v, want %+v", entries[1], want1)
	}

	// xgs/6/1 and xgs/6/2 both exist, proving two rows on the same PON
	// port (different ONU index) both parse — exactly the data
	// NextFreeIndex needs.
	if entries[2].Interface != "xgs/6/1" || entries[3].Interface != "xgs/6/2" {
		t.Errorf("entries[2].Interface, entries[3].Interface = %q, %q, want xgs/6/1, xgs/6/2",
			entries[2].Interface, entries[3].Interface)
	}
}

// TestParseONUSummaryEntriesEmptyInput proves the parser does not error
// or panic on the total absence of any table — this codebase has no
// confirmed sample of an OLT with zero provisioned ONUs, so this only
// covers the degenerate case, not a confirmed real shape.
func TestParseONUSummaryEntriesEmptyInput(t *testing.T) {
	entries, err := kontron.ParseONUSummaryEntries("")
	if err != nil {
		t.Fatalf("ParseONUSummaryEntries() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("len(entries) = %d, want 0", len(entries))
	}
}

// TestParseONUSummaryEntriesStopsAtBlankLine is synthesized input (not a
// real capture) proving a blank line after the last data row ends
// parsing rather than producing a spurious entry from it — see
// ParseONUSummaryEntries' own doc comment on why "ends at a blank line"
// is this parser's (unverified) assumption about where the real table
// actually stops.
func TestParseONUSummaryEntriesStopsAtBlankLine(t *testing.T) {
	withTrailer := realONUSummarySample + "\nsome shell noise after the table\n"
	entries, err := kontron.ParseONUSummaryEntries(withTrailer)
	if err != nil {
		t.Fatalf("ParseONUSummaryEntries() error = %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("len(entries) = %d, want 4 (trailer after the blank line should be ignored)", len(entries))
	}
}
