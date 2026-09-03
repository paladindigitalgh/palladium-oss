package workflow

import "github.com/paladindigitalgh/palladium-oss/internal/plugin"

// Definition is a Workflow Definition (docs/05-WORKFLOW-ENGINE.md,
// "Workflow Definition"): the blueprint for an operational task,
// versioned and immutable once published. v1 keeps this to its simplest
// honest shape — a definition names exactly one plugin.Capability to
// invoke — rather than a general step sequence; docs/05 section 18
// explicitly defers "conditional branching," "parallel execution," and a
// "visual workflow designer" to future work, so there is nothing here to
// build toward yet.
type Definition struct {
	Name       string
	Version    int
	Capability plugin.Capability
}

// Definitions is the authoritative, in-code registry of every Workflow
// Definition v1 supports, keyed by Name. A definition-authoring UI or
// database table is explicitly out of scope (see this type's doc
// comment) — publishing a new version means adding a new entry here,
// which is what "immutable once published" means for a hardcoded set of
// definitions.
var Definitions = map[string]Definition{
	"provision-service":   {Name: "provision-service", Version: 1, Capability: plugin.ProvisionService},
	"reprovision-service": {Name: "reprovision-service", Version: 1, Capability: plugin.ReprovisionService},
	"suspend-service":     {Name: "suspend-service", Version: 1, Capability: plugin.SuspendService},
	"resume-service":      {Name: "resume-service", Version: 1, Capability: plugin.ResumeService},
	"disconnect-service":  {Name: "disconnect-service", Version: 1, Capability: plugin.DisconnectService},
	"synchronize-service": {Name: "synchronize-service", Version: 1, Capability: plugin.SynchronizeService},
}
