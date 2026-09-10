package kontron

import (
	"fmt"
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

// ParsePortNumber extracts a PON port's number from a fully-assigned
// Kontron interface string of the form "<prefix>/<port>/<index>" (e.g.
// "xgs/1/1" -> 1) -- the exact inverse of how AuthorizeONU builds one
// (port + "/" + index, see that method's own implementation). Used to
// resolve which internal/ponport.PONPort a freshly authorized ONU's
// interface belongs to, so internal/provisioning/kontron/service can
// keep the Access Network topology (internal/ponport,
// internal/accessinterface) in sync with what it already knows from a
// real OLT authorization, without an operator ever having to enter that
// topology by hand for a Device authorized this way.
//
// Returns an error for anything that does not parse as exactly three
// "/"-separated segments with a positive integer in the middle one --
// deliberately strict, unlike NextFreeIndex's lenient "silently ignore
// what does not parse" stance above: that function is scanning read-only
// diagnostic data for a best-effort hint, while this one is about to
// create real inventory records and must not guess.
func ParsePortNumber(iface string) (int, error) {
	parts := strings.Split(iface, "/")
	if len(parts) != 3 {
		return 0, fmt.Errorf("kontron: interface %q does not have the expected <prefix>/<port>/<index> shape", iface)
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil || port <= 0 {
		return 0, fmt.Errorf("kontron: interface %q does not have a valid port number", iface)
	}
	return port, nil
}
