package tunnel

import (
	"testing"
	"time"
)

func TestTunnelRXWaitMS(t *testing.T) {
	if got := tunnelRXWaitMS(true, 200*time.Millisecond); got != 500 {
		t.Fatalf("with TX: got %d ms, want 500", got)
	}
	if got := tunnelRXWaitMS(false, 10*time.Millisecond); got != 500 {
		t.Fatalf("idle clamp min: got %d ms, want 500", got)
	}
	if got := tunnelRXWaitMS(false, 5*time.Second); got != 3000 {
		t.Fatalf("idle clamp max: got %d ms, want 3000", got)
	}
}
