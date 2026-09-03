// Package plugin implements Palladium's Plugin Architecture (v1): the
// capability-driven contract between the core platform and vendor-
// specific implementations (see docs/06-PLUGIN-ARCHITECTURE.md). The
// core never depends on a concrete vendor — only on this package's
// interfaces — so a new vendor can be supported by implementing Plugin,
// never by changing internal/workflow or anything upstream of it.
package plugin

import (
	"github.com/paladindigitalgh/palladium-oss/internal/service"
	"github.com/paladindigitalgh/palladium-oss/internal/serviceequipment"
)

// Resource is what internal/workflow's engine passes to a Plugin's
// Execute: the Service being acted on and the single piece of Service
// Equipment this particular call concerns. The engine calls Execute once
// per active ServiceEquipment record for a Service, so Resource carries
// one Equipment, not a slice — a Plugin answers "what do I do for this
// one device," never "what do I do for this whole service."
type Resource struct {
	Service   service.Service
	Equipment serviceequipment.ServiceEquipment
}
