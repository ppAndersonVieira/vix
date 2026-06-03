package telemetry

import (
	"os"
	"testing"

	"github.com/zalando/go-keyring"
)

func init() {
	// Use in-memory mock keyring for all tests
	keyring.MockInit()
}

func TestGetOrCreateDeviceID_CreatesNew(t *testing.T) {
	// Clean slate
	keyring.MockInit()

	id, err := GetOrCreateDeviceID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty device ID")
	}
	if len(id) != 36 { // UUID format: 8-4-4-4-12
		t.Fatalf("expected UUID format, got %q", id)
	}
}

func TestGetOrCreateDeviceID_ReadsExisting(t *testing.T) {
	keyring.MockInit()

	// Create first
	id1, err := GetOrCreateDeviceID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read again
	id2, err := GetOrCreateDeviceID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if id1 != id2 {
		t.Fatalf("expected same device ID, got %q and %q", id1, id2)
	}
}

func TestOptOut_EnvVar(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"off", true},
		{"false", true},
		{"0", true},
		{"OFF", true},
		{"False", true},
		{"", false},
		{"on", false},
		{"1", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			os.Setenv("VIX_TELEMETRY", tt.value)
			defer os.Unsetenv("VIX_TELEMETRY")

			got := isOptedOut()
			if got != tt.expected {
				t.Errorf("VIX_TELEMETRY=%q: isOptedOut()=%v, want %v", tt.value, got, tt.expected)
			}
		})
	}
}

func TestHashWorkflowName(t *testing.T) {
	h1 := HashWorkflowName("Plan Workflow")
	h2 := HashWorkflowName("Plan Workflow")
	h3 := HashWorkflowName("Review Workflow")

	if h1 != h2 {
		t.Errorf("expected deterministic hash, got %q and %q", h1, h2)
	}
	if h1 == h3 {
		t.Error("expected different hashes for different names")
	}
	if len(h1) != 12 {
		t.Errorf("expected 12 hex chars, got %d: %q", len(h1), h1)
	}
}

func TestTrack_NoopWhenDisabled(t *testing.T) {
	// enabled is false by default (no Init called)
	enabled = false
	// Should not panic
	Track("test_event", map[string]interface{}{"key": "value"})
}

func TestCaptureException_NoopWhenDisabled(t *testing.T) {
	// enabled is false by default (no Init called); client is nil, so this
	// must short-circuit without dereferencing it.
	enabled = false
	// Should not panic
	CaptureException("panic", "boom")
}

func TestTrackPanic_NoopWhenDisabled(t *testing.T) {
	enabled = false
	// Should not panic even with a nil client.
	TrackPanic("test.site", "boom", []byte("goroutine stack"))
}
