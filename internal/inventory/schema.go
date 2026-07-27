package inventory

// Document describes the inventory hierarchy for introspection over HTTP.
// It is not a database schema — per CLAUDE.md, schema changes happen only
// through migrations — it exists purely so the shape of this domain can be
// verified while no CRUD endpoints exist yet.
type Document struct {
	Root         string              `json:"root"`
	Children     map[string][]string `json:"children"`
	DeviceFields []string            `json:"deviceFields"`
}

// deviceFields lists the notable Device fields for the temporary
// /inventory/schema endpoint. It is a curated list, not a full field dump:
// it omits structural fields (ID, CreatedAt, UpdatedAt, the nullable
// RackID) and shows only what a caller would set to describe a device,
// matching the milestone's goal of documenting what changed. Keep this in
// sync with the Device struct in model.go if its fields change.
var deviceFields = []string{
	"name",
	"manufacturer",
	"model",
	"serialNumber",
	"assetTag",
	"status",
}

// Hierarchy returns the current inventory hierarchy — Site -> Building ->
// Room -> Rack -> Device — plus the notable Device fields. Keep the
// hierarchy in sync with the parent references in model.go if it ever
// changes.
func Hierarchy() Document {
	return Document{
		Root: "site",
		Children: map[string][]string{
			"site":     {"building"},
			"building": {"room"},
			"room":     {"rack"},
			"rack":     {"device"},
		},
		DeviceFields: deviceFields,
	}
}
