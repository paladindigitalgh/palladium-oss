package inventory_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/inventory"
)

func TestHandlerSchemaReturnsHierarchyAsJSON(t *testing.T) {
	h := inventory.NewHandler()

	rec := httptest.NewRecorder()
	h.Schema(rec, httptest.NewRequest(http.MethodGet, "/api/v1/inventory/schema", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var got inventory.Document
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if got.Root != "site" {
		t.Errorf("root = %q, want site", got.Root)
	}
	if children := got.Children["rack"]; len(children) != 1 || children[0] != "device" {
		t.Errorf("children[rack] = %v, want [device]", children)
	}

	wantDeviceFields := []string{"name", "manufacturer", "model", "serialNumber", "assetTag", "status"}
	if !reflect.DeepEqual(got.DeviceFields, wantDeviceFields) {
		t.Errorf("deviceFields = %v, want %v", got.DeviceFields, wantDeviceFields)
	}
}
