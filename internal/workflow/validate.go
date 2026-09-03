package workflow

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/validate"
)

// Validate reports whether i has every required field set: a present
// ServiceID, a DefinitionName naming a real Definition (see
// definition.go), and a Status that is one of its defined values. It
// never checks whether Status is a legal transition from whatever was
// previously persisted — that requires a repository round trip and
// belongs to internal/workflow/service.Service.
func (i Instance) Validate() error {
	errs := validate.New()

	if i.ServiceID == uuid.Nil {
		errs.Add("service_id", "is required")
	}
	if _, ok := Definitions[i.DefinitionName]; !ok {
		errs.Add("definition_name", fmt.Sprintf("must be one of: %s", definitionNames()))
	}
	if !i.Status.Valid() {
		errs.Add("status", fmt.Sprintf("must be one of: %s", statusNames()))
	}

	return errs.Err()
}

func definitionNames() string {
	names := make([]string, 0, len(Definitions))
	for name := range Definitions {
		names = append(names, name)
	}
	return fmt.Sprint(names)
}
