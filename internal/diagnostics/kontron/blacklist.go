package kontron

import (
	"fmt"
	"strconv"
	"strings"
)

// BlacklistEntry is one row of BlacklistedONUs' output, parsed into its
// four columns.
//
// This package's own doc comment establishes "no parsing, return raw
// text verbatim" as this file's neighbors' shared rule. BlacklistEntry
// and ParseBlacklistEntries are a deliberate, narrow exception to it,
// not a reversal of it: every other command here produces text meant for
// an operator to read on a terminal, but a blacklist entry's serial
// number is meant to become a selectable option in a picker UI (an
// operator choosing which physically-detected ONU to bring into
// service) — that requires structured fields, not a verbatim string, so
// parsing has to happen somewhere, and this is the one place close
// enough to the exact table format to own it.
type BlacklistEntry struct {
	Interface      string
	SerialNumber   string
	RegistrationID string
	Cause          string
}

// The real, captured sample this parser is built from, for reference
// (also see blacklist_test.go, which asserts against it verbatim):
//
//	Interface  Serial Number     Password/Registration Id                Cause
//	---------  ----------------  --------------------------------------  -----------------------
//	xgs/6      ISKT2308DD88      ""                                      Serial Number not known
//	---------------------------
//	Total: 1
//
// ParseBlacklistEntries has exactly this one real sample behind it. What
// is unverified, pending real OLT access, is called out below.

// columnRuleLineFields splits line on runs of spaces, returning nil
// unless every field is non-empty and made entirely of '-' and there are
// at least two of them — the shape of the header's column-boundary line
// (e.g. "---------  ----------------  ...") but not the single-run
// closing footer line (e.g. "---------------------------"), which has
// only one field. Distinguishing the two by field count, not by length
// or position, is what lets this survive a differently-sized response
// (a longer serial number, a longer cause string) without becoming
// mis-tuned.
func columnRuleLineFields(line string) []string {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return nil
	}
	for _, f := range fields {
		if strings.Trim(f, "-") != "" {
			return nil
		}
	}
	return fields
}

// columnStarts returns the byte offset, within line, that each field
// returned by columnRuleLineFields starts at — the column boundaries a
// data row is sliced on. Computed by walking line for each field's
// substring, in order, rather than assuming fixed widths, since this
// method's whole point is to adapt to whatever column widths a given
// response actually used.
func columnStarts(line string, fields []string) []int {
	starts := make([]int, len(fields))
	pos := 0
	for i, f := range fields {
		idx := strings.Index(line[pos:], f)
		pos += idx
		starts[i] = pos
		pos += len(f)
	}
	return starts
}

// sliceColumn returns the trimmed text of line from starts[i] up to
// starts[i+1] (or end of line for the last column, since Cause is free
// text with no fixed width of its own) — extending each column to the
// next one's start, not stopping at its own header's width, so a value
// wider than its column header (a long cause string, a long serial
// number) does not get truncated as long as it does not overrun into the
// next column's data.
func sliceColumn(line string, starts []int, i int) string {
	if i >= len(starts) || starts[i] >= len(line) {
		return ""
	}
	end := len(line)
	if i+1 < len(starts) && starts[i+1] <= len(line) {
		end = starts[i+1]
	}
	return strings.TrimSpace(line[starts[i]:end])
}

// ParseBlacklistEntries parses BlacklistedONUs' raw output into one
// BlacklistEntry per detected-but-unauthorized ONU.
//
// Column boundaries are computed fresh from each response's own header
// rule line (see columnRuleLineFields/columnStarts), not hardcoded, so a
// response with a wider serial number or cause string than the one real
// sample this was built from still parses correctly.
//
// The closing "Total: N" line is used as a sanity check, not just
// ignored: if it disagrees with the number of rows actually parsed, that
// means something about this response's column layout was not what this
// parser assumed (data drifted across a column boundary it guessed
// wrong), and returning a wrong-but-plausible-looking list to a picker
// UI is worse than surfacing an error — the caller (see
// internal/diagnostics/kontron/service.KontronService.AggregatedBlacklist)
// treats that the same as an unreachable OLT: skip it, report why, keep
// going for every other OLT.
//
// Unverified, pending real OLT access: what this command prints when
// there are zero blacklisted ONUs. It might still print the header and
// rule line with nothing under them and "Total: 0", or it might omit the
// table entirely. This function is deliberately lenient about that
// unknown: if no column rule line is found at all, it returns an empty
// slice with no error, treating "nothing parseable" the same as
// "confirmed zero entries" rather than failing. The tradeoff this
// accepts: a genuinely unexpected or malformed response that also never
// reaches a "Total:" line (so the sanity check above never fires) would
// silently read as zero entries rather than surfacing as a parse error.
// Given this package's own design goal for this feature — populate what
// can be found across every OLT, never let one bad response block the
// rest — that is the right side to err on, but it does mean this one
// edge case needs confirming against a real C16 before being trusted as
// "definitely means zero," not "unexpected output silently ignored."
func ParseBlacklistEntries(raw string) ([]BlacklistEntry, error) {
	var starts []int
	var entries []BlacklistEntry

	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")

		if fields := columnRuleLineFields(line); fields != nil {
			starts = columnStarts(line, fields)
			continue
		}

		if starts == nil {
			// Still in the banner/header, before the column rule line —
			// or this response has no table at all (see the doc comment
			// above on why that reads as zero entries, not an error).
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if after, ok := strings.CutPrefix(trimmed, "Total:"); ok {
			total, err := strconv.Atoi(strings.TrimSpace(after))
			if err != nil {
				return nil, fmt.Errorf("kontron: parse blacklist: unreadable %q line", trimmed)
			}
			if total != len(entries) {
				return nil, fmt.Errorf("kontron: parse blacklist: %q reports %d but %d rows were parsed", trimmed, total, len(entries))
			}
			break
		}
		if strings.Trim(trimmed, "-") == "" {
			// The closing footer rule (a single contiguous dash run,
			// distinct from the header's column rule line — see
			// columnRuleLineFields). Skip it rather than stopping here:
			// the "Total:" sanity check above still needs to see the
			// line that (usually) follows it.
			continue
		}

		entries = append(entries, BlacklistEntry{
			Interface:      sliceColumn(line, starts, 0),
			SerialNumber:   sliceColumn(line, starts, 1),
			RegistrationID: sliceColumn(line, starts, 2),
			Cause:          sliceColumn(line, starts, 3),
		})
	}

	return entries, nil
}
