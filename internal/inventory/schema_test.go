package inventory_test

import (
	"reflect"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

func TestHierarchyMatchesDocumentedShape(t *testing.T) {
	got := inventory.Hierarchy()

	if got.Root != "site" {
		t.Errorf("Root = %q, want %q", got.Root, "site")
	}

	wantChildren := map[string][]string{
		"site":     {"building"},
		"building": {"room"},
		"room":     {"rack"},
		"rack":     {"device"},
	}
	if !reflect.DeepEqual(got.Children, wantChildren) {
		t.Errorf("Children = %v, want %v", got.Children, wantChildren)
	}

	wantDeviceFields := []string{"name", "manufacturer", "model", "serialNumber", "assetTag", "status"}
	if !reflect.DeepEqual(got.DeviceFields, wantDeviceFields) {
		t.Errorf("DeviceFields = %v, want %v", got.DeviceFields, wantDeviceFields)
	}
}
