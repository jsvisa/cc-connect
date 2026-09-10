package chatapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chenhg5/cc-connect/core"
)

type fakeQueuePurger struct {
	got     []string
	purged  bool
	message string
	user    string
	channel string
}

func (f *fakeQueuePurger) BindQueuedMessagePurger(qp core.QueuedMessagePurger) {}
func (f *fakeQueuePurger) PurgeQueuedMessage(messageID, user, channel string) bool {
	f.got = append(f.got, messageID)
	if messageID == f.message && user == f.user && channel == f.channel {
		f.purged = true
		return true
	}
	return false
}

// A run that ended at message_queued time is no longer in pending; cancelling
// it must withdraw the still-waiting queued message through the engine.
func TestCancelRunWithdrawsQueuedMessage(t *testing.T) {
	p := newTestPlatform(t, map[string]any{"token": "secret", "sse_ping_interval": "0s"})
	fake := &fakeQueuePurger{message: "run_q1", user: "user_001", channel: testChannel}
	p.BindQueuedMessagePurger(fake)

	req := httptest.NewRequest(http.MethodPost, "/v1/runs/run_q1/cancel", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-Chat-API-User", "user_001")
	req.Header.Set("X-Chat-API-Channel", testChannel)
	rec := httptest.NewRecorder()
	p.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"cancelled":"queued"`) {
		t.Fatalf("missing cancelled=queued marker: %s", rec.Body.String())
	}
	if len(fake.got) != 1 || fake.got[0] != "run_q1" {
		t.Fatalf("purger not called with run id: %v", fake.got)
	}

	// Mismatched user/channel must not purge someone else's queued message.
	req2 := httptest.NewRequest(http.MethodPost, "/v1/runs/run_q1/cancel", nil)
	req2.Header.Set("Authorization", "Bearer secret")
	req2.Header.Set("X-Chat-API-User", "someone_else")
	req2.Header.Set("X-Chat-API-Channel", testChannel)
	rec2 := httptest.NewRecorder()
	p.routes().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotFound {
		t.Fatalf("mismatched user status=%d, want 404", rec2.Code)
	}
}

// Without a bound purger (engine not wired) the cancel stays a clean 404.
func TestCancelRunWithoutPurgerReturnsNotFound(t *testing.T) {
	p := newTestPlatform(t, map[string]any{"token": "secret"})
	p.queuePurger = nil

	req := httptest.NewRequest(http.MethodPost, "/v1/runs/run_none/cancel", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("X-Chat-API-User", "user_001")
	req.Header.Set("X-Chat-API-Channel", testChannel)
	rec := httptest.NewRecorder()
	p.routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404 when nothing matches", rec.Code)
	}
	_ = http.MethodPost
	_ = strings.TrimSpace
}
