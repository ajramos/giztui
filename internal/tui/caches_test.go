package tui

import (
	"testing"

	"github.com/ajramos/giztui/internal/gmail"
)

func TestAppCachesMessageRoundTrip(t *testing.T) {
	c := newAppCaches()

	if _, ok := c.messageGet("missing"); ok {
		t.Fatalf("expected miss for absent message id")
	}

	msg := &gmail.Message{}
	c.messageSet("id1", msg)

	got, ok := c.messageGet("id1")
	if !ok {
		t.Fatalf("expected hit for set message id")
	}
	if got != msg {
		t.Fatalf("expected same message pointer back")
	}
}

func TestAppCachesRenderRoundTrip(t *testing.T) {
	c := newAppCaches()

	if _, ok := c.renderGet("k"); ok {
		t.Fatalf("expected miss for absent render key")
	}

	c.renderSet("k", "value")

	got, ok := c.renderGet("k")
	if !ok || got != "value" {
		t.Fatalf("expected render hit with value, got %q ok=%v", got, ok)
	}
}

func TestAppCachesRenderLen(t *testing.T) {
	c := newAppCaches()
	c.renderSet("a", "1")
	c.renderSet("b", "2")
	if n := c.renderLen(); n != 2 {
		t.Fatalf("expected render len 2, got %d", n)
	}
	c.renderDelete("a")
	if n := c.renderLen(); n != 1 {
		t.Fatalf("expected render len 1 after delete, got %d", n)
	}
}

func TestAppCachesInviteGetSetDelete(t *testing.T) {
	c := newAppCaches()

	if _, ok := c.inviteGet("none"); ok {
		t.Fatalf("expected miss for absent invite")
	}

	inv := Invite{UID: "u1", Summary: "Meeting"}
	c.inviteSet("m1", inv)

	got, ok := c.inviteGet("m1")
	if !ok || got.UID != "u1" {
		t.Fatalf("expected invite hit with UID u1, got %+v ok=%v", got, ok)
	}

	if c.inviteLen() != 1 {
		t.Fatalf("expected invite len 1, got %d", c.inviteLen())
	}

	c.inviteDelete("m1")
	if _, ok := c.inviteGet("m1"); ok {
		t.Fatalf("expected miss after invite delete")
	}
}

func TestAppCachesInviteFindAnyWithUID(t *testing.T) {
	c := newAppCaches()

	if _, ok := c.inviteFindAnyWithUID(); ok {
		t.Fatalf("expected no invite found in empty cache")
	}

	c.inviteSet("a", Invite{UID: ""})
	if _, ok := c.inviteFindAnyWithUID(); ok {
		t.Fatalf("expected no invite found when only empty-UID invites present")
	}

	c.inviteSet("b", Invite{UID: "has-uid"})
	got, ok := c.inviteFindAnyWithUID()
	if !ok || got.UID != "has-uid" {
		t.Fatalf("expected to find invite with UID, got %+v ok=%v", got, ok)
	}
}

func TestAppCachesAIInFlight(t *testing.T) {
	c := newAppCaches()

	if c.aiInFlightHas("x") {
		t.Fatalf("expected no in-flight for absent id")
	}

	c.aiInFlightSet("x")
	if !c.aiInFlightHas("x") {
		t.Fatalf("expected in-flight after set")
	}

	c.aiInFlightDelete("x")
	if c.aiInFlightHas("x") {
		t.Fatalf("expected no in-flight after delete")
	}
}

func TestAppCachesAIInFlightCancelFirst(t *testing.T) {
	c := newAppCaches()

	if c.aiInFlightCancelFirst() {
		t.Fatalf("expected no active in-flight to cancel in empty cache")
	}

	c.aiInFlightSet("a")
	if !c.aiInFlightCancelFirst() {
		t.Fatalf("expected to cancel an active in-flight")
	}
	// After cancellation, the entry should be marked false (not active).
	if c.aiInFlightHas("a") {
		t.Fatalf("expected in-flight 'a' to be inactive after cancel")
	}
}
