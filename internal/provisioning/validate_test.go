package provisioning_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/paladindigitalgh/palladium-oss/internal/platform/apperror"
	"github.com/paladindigitalgh/palladium-oss/internal/provisioning"
)

// assertInvalid mirrors internal/serviceequipment/validate_test.go's
// helper of the same name: every domain package's Validate() must return
// an *apperror.Error of KindInvalid.
func assertInvalid(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("Validate() = nil, want error")
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("Validate() error is not an *apperror.Error: %v", err)
	}
	if appErr.Kind != apperror.KindInvalid {
		t.Errorf("Kind = %q, want %q", appErr.Kind, apperror.KindInvalid)
	}
}

func validProvisioningJob() provisioning.ProvisioningJob {
	return provisioning.ProvisioningJob{
		ServiceID: uuid.New(),
		Operation: provisioning.ProvisioningOperationProvision,
		Status:    provisioning.ProvisioningStatusPending,
	}
}

func TestProvisioningJobValidate(t *testing.T) {
	if err := validProvisioningJob().Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}

	assertInvalid(t, provisioning.ProvisioningJob{}.Validate())
}

func TestProvisioningJobValidateRequiresServiceID(t *testing.T) {
	j := validProvisioningJob()
	j.ServiceID = uuid.Nil

	assertInvalid(t, j.Validate())
}

func TestProvisioningJobValidateRequiresKnownOperation(t *testing.T) {
	unrecognized := validProvisioningJob()
	unrecognized.Operation = provisioning.ProvisioningOperation("Deprovision")
	assertInvalid(t, unrecognized.Validate())

	unset := validProvisioningJob()
	unset.Operation = ""
	assertInvalid(t, unset.Validate())

	for _, o := range []provisioning.ProvisioningOperation{
		provisioning.ProvisioningOperationProvision,
		provisioning.ProvisioningOperationReprovision,
		provisioning.ProvisioningOperationSuspend,
		provisioning.ProvisioningOperationResume,
		provisioning.ProvisioningOperationDisconnect,
		provisioning.ProvisioningOperationSynchronize,
	} {
		j := validProvisioningJob()
		j.Operation = o
		if err := j.Validate(); err != nil {
			t.Errorf("Validate() (operation %q) = %v, want nil", o, err)
		}
	}
}

func TestProvisioningJobValidateRequiresKnownStatus(t *testing.T) {
	unrecognized := validProvisioningJob()
	unrecognized.Status = provisioning.ProvisioningStatus("Cancelling")
	assertInvalid(t, unrecognized.Validate())

	unset := validProvisioningJob()
	unset.Status = ""
	assertInvalid(t, unset.Validate())

	for _, s := range []provisioning.ProvisioningStatus{
		provisioning.ProvisioningStatusPending,
		provisioning.ProvisioningStatusRunning,
		provisioning.ProvisioningStatusSucceeded,
		provisioning.ProvisioningStatusFailed,
		provisioning.ProvisioningStatusCancelled,
	} {
		j := validProvisioningJob()
		j.Status = s
		if err := j.Validate(); err != nil {
			t.Errorf("Validate() (status %q) = %v, want nil", s, err)
		}
	}
}

func TestProvisioningJobValidateOptionalFieldsAreOptional(t *testing.T) {
	j := validProvisioningJob() // no RequestedByUserID, ErrorMessage, StartedAt, CompletedAt
	if err := j.Validate(); err != nil {
		t.Errorf("Validate() (no optional fields) = %v, want nil", err)
	}
}

// TestProvisioningJobValidateDoesNotCheckStateTransitions proves goal 4's
// explicit instruction: Validate is a pure, single-record check with no
// knowledge of any prior persisted Status — a ProvisioningJob claiming
// Status Succeeded is well-formed on its own regardless of what state
// machine rules might apply to how it got there.
func TestProvisioningJobValidateDoesNotCheckStateTransitions(t *testing.T) {
	j := validProvisioningJob()
	j.Status = provisioning.ProvisioningStatusSucceeded // not reachable from nothing, but Validate does not know that

	if err := j.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil; Validate must not perform state-transition validation", err)
	}
}
