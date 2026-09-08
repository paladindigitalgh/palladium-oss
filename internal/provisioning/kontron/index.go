package kontron

import (
	"strconv"
	"strings"

	diagnosticskontron "github.com/paladindigitalgh/palladium-oss/internal/diagnostics/kontron"
)

// NextFreeIndex returns the lowest unused positive ONU index on port
// (e.g. "xgs/6"), given every currently-known
// diagnosticskontron.ONUSummaryEntry across the whole chassis (see
// internal/diagnostics/kontron.ParseONUSummaryEntries).
//
// Gaps are filled, not skipped: if indices 1 and 3 are in use on port,
// this returns 2, not 4 — a deliberate product decision (an operator
// wants a freed-up slot reused before the range keeps growing), not an
// incidental effect of how this is implemented.
//
// Entries for other ports, or whose Interface does not parse as
// "<port>/<int>", are silently ignored: this is a read used to make a
// provisioning decision about one specific port, not a validation of
// the device's own data, so a malformed or unrelated row must never
// block finding a free slot on the port that is actually being
// provisioned.
func NextFreeIndex(entries []diagnosticskontron.ONUSummaryEntry, port string) int {
	used := make(map[int]bool)
	prefix := port + "/"
	for _, e := range entries {
		suffix, ok := strings.CutPrefix(e.Interface, prefix)
		if !ok {
			continue
		}
		index, err := strconv.Atoi(suffix)
		if err != nil || index <= 0 {
			continue
		}
		used[index] = true
	}

	for i := 1; ; i++ {
		if !used[i] {
			return i
		}
	}
}
