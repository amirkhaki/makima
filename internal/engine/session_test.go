package engine

import (
	"testing"
	"time"
)

func TestSessionGracePeriod(t *testing.T) {
	session := NewSession("test-key", 100*time.Millisecond, 200*time.Millisecond)

	if !session.InGrace() {
		t.Fatal("session should be in grace period immediately after creation")
	}

	time.Sleep(50 * time.Millisecond)

	if !session.InGrace() {
		t.Fatal("session should still be in grace period before grace expires")
	}
}

func TestSessionCooldown(t *testing.T) {
	session := NewSession("test-key", 0, 100*time.Millisecond)

	session.FireAction()

	if !session.InCooldown() {
		t.Fatal("session should be in cooldown after firing action")
	}

	time.Sleep(50 * time.Millisecond)

	if !session.InCooldown() {
		t.Fatal("session should still be in cooldown before cooldown expires")
	}

	time.Sleep(60 * time.Millisecond)

	if session.InCooldown() {
		t.Fatal("session should not be in cooldown after cooldown expires")
	}
}

func TestSessionGraceExpires(t *testing.T) {
	session := NewSession("test-key", 50*time.Millisecond, 200*time.Millisecond)

	if !session.InGrace() {
		t.Fatal("session should be in grace period immediately after creation")
	}

	time.Sleep(60 * time.Millisecond)

	if session.InGrace() {
		t.Fatal("session should not be in grace period after grace expires")
	}
}

func TestSessionKey(t *testing.T) {
	session := NewSession("my-key", 100*time.Millisecond, 200*time.Millisecond)

	if session.Key() != "my-key" {
		t.Errorf("expected key 'my-key', got '%s'", session.Key())
	}
}

func TestSessionManagerGetOrCreate(t *testing.T) {
	mgr := NewSessionManager()

	session1 := mgr.GetOrCreate("key1", 100*time.Millisecond, 200*time.Millisecond)
	session2 := mgr.GetOrCreate("key1", 100*time.Millisecond, 200*time.Millisecond)

	if session1 != session2 {
		t.Fatal("GetOrCreate should return the same session for the same key")
	}

	session3 := mgr.GetOrCreate("key2", 100*time.Millisecond, 200*time.Millisecond)

	if session1 == session3 {
		t.Fatal("GetOrCreate should return different sessions for different keys")
	}
}

func TestSessionNoGraceNoCooldown(t *testing.T) {
	session := NewSession("test-key", 0, 0)

	if session.InGrace() {
		t.Fatal("session should not be in grace period when grace is 0")
	}

	session.FireAction()

	if session.InCooldown() {
		t.Fatal("session should not be in cooldown when cooldown is 0")
	}
}
