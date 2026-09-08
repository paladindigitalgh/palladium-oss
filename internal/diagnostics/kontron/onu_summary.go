package kontron

import "strings"

// ONUSummaryEntry is one row of ONUSummary's output, parsed into its ten
// columns.
//
// Like BlacklistEntry (see blacklist.go's own doc comment for the full
// reasoning this one shares), this is a deliberate, narrow exception to
// this file's "no parsing" rule: unlike every other command's output,
// which is meant for an operator to read on a terminal, ONUSummary's
// output is also needed to make a provisioning decision — specifically,
// which ONU indices are already in use on a given PON port (see
// internal/provisioning/kontron.NextFreeIndex) — and that requires
// structured fields, not a verbatim string.
type ONUSummaryEntry struct {
	Interface      string
	OperState      string
	AdminState     string
	ONUState       string
	ServiceState   string
	SerialNumber   string
	RegistrationID string
	IPAddress      string
	MACAddress     string
	Description    string
}

// The real, captured sample this parser is built from (truncated here;
// see onu_summary_test.go for the exact rows asserted against verbatim):
//
//	jamestown-steele-olt-03#show onu int all
//	ONU       Oper     Admin    ONU               Service  Serial
//	Interface State    State    State             State    Number       Password/Registration Id               IP Address      MAC Address       Description
//	--------- -------- -------- ----------------  -------- ------------ -------------------------------------- --------------- ----------------- ------------------
//	xgs/1/1   Up       Enable   Active            Enable   ISKT230660D0 ""                                     10.70.180.25    48:55:41:06:60:D0 ONU xgs/1/1
//	xgs/3/1   Down     Enable   Not Configured    Enable   ISKT23066250 ""                                     0.0.0.0         00:00:00:00:00:00 ONU xgs/3/1
//
// Two things distinguish this table from BlacklistedONUs' (see
// blacklist.go): the column titles span two text lines before the
// column rule line, not one (this parser does not care — it only looks
// for the rule line, never reads the header text), and — the one thing
// genuinely unverified, pending real OLT access — this real sample has
// no closing "Total: N" line and no closing footer dash rule; the table
// simply ends after the last data row, with nothing following in the
// captured output. ParseONUSummaryEntries is built to stop at the first
// blank line or end of input, since that is what the one real sample
// available actually does; if a real C16 turns out to print something
// else after the last row (a footer, a second table, shell noise before
// the prompt reappears), that will need confirming and this parser may
// need a small adjustment.
//
// The returned error is always nil today — unlike ParseBlacklistEntries,
// there is no "Total: N" line here to sanity-check row counts against.
// The signature still returns one, matching ParseBlacklistEntries'
// shape, so a future sanity check (or the "what happens after the last
// row" question above resolving into one) is an additive change for
// every caller, not a breaking one.
func ParseONUSummaryEntries(raw string) ([]ONUSummaryEntry, error) {
	var starts []int
	var entries []ONUSummaryEntry

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")

		if fields := columnRuleLineFields(line); fields != nil {
			starts = columnStarts(line, fields)
			continue
		}

		if starts == nil {
			// Still in the two-line text header, before the column rule
			// line.
			continue
		}

		if strings.TrimSpace(line) == "" {
			// See this function's own doc comment: the one real sample
			// behind this parser ends the table with nothing following,
			// so a blank line (or end of input, via strings.Split's own
			// behavior) is treated as "no more rows," not an error.
			break
		}

		entries = append(entries, ONUSummaryEntry{
			Interface:      sliceColumn(line, starts, 0),
			OperState:      sliceColumn(line, starts, 1),
			AdminState:     sliceColumn(line, starts, 2),
			ONUState:       sliceColumn(line, starts, 3),
			ServiceState:   sliceColumn(line, starts, 4),
			SerialNumber:   sliceColumn(line, starts, 5),
			RegistrationID: sliceColumn(line, starts, 6),
			IPAddress:      sliceColumn(line, starts, 7),
			MACAddress:     sliceColumn(line, starts, 8),
			Description:    sliceColumn(line, starts, 9),
		})
	}

	return entries, nil
}
