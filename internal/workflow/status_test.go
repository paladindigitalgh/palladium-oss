package workflow_test

import (
	"testing"

	"github.com/paladindigitalgh/palladium-oss/internal/workflow"
)

func TestStatusCanTransitionTo(t *testing.T) {
	cases := []struct {
		from, to workflow.Status
		want     bool
	}{
		{workflow.StatusPending, workflow.StatusRunning, true},
		{workflow.StatusPending, workflow.StatusCancelled, true},
		{workflow.StatusPending, workflow.StatusSucceeded, false},
		{workflow.StatusRunning, workflow.StatusSucceeded, true},
		{workflow.StatusRunning, workflow.StatusFailed, true},
		{workflow.StatusFailed, workflow.StatusPending, true},
		{workflow.StatusSucceeded, workflow.StatusRunning, false},
		{workflow.StatusCancelled, workflow.StatusRunning, false},
	}

	for _, c := range cases {
		if got := c.from.CanTransitionTo(c.to); got != c.want {
			t.Errorf("%s.CanTransitionTo(%s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestStatusValid(t *testing.T) {
	if !workflow.StatusPending.Valid() {
		t.Error("StatusPending.Valid() = false, want true")
	}
	if workflow.Status("bogus").Valid() {
		t.Error(`Status("bogus").Valid() = true, want false`)
	}
}
