package core

import (
	"testing"
)

// Purging a queued message must remove exactly the matching entry (message id
// + user + channel), leaving the rest of the queue intact.
func TestPurgeQueuedMessage(t *testing.T) {
	e := NewEngine("test", &stubAgent{}, []Platform{&stubPlatformEngine{n: "test"}}, "", LangEnglish)
	key := "chat-api:chan:conv_1"
	state := &interactiveState{
		agentSession: &stubAgentSession{},
		pendingMessages: []queuedMessage{
			{messageID: "run_a", userID: "u1", channelKey: "chan"},
			{messageID: "run_b", userID: "u1", channelKey: "chan"},
			{messageID: "run_c", userID: "u2", channelKey: "chan"},
		},
	}
	e.interactiveMu.Lock()
	e.interactiveStates[key] = state
	e.interactiveMu.Unlock()

	if !e.PurgeQueuedMessage("run_b", "u1", "chan") {
		t.Fatal("matching queued message must purge")
	}
	if got := len(e.interactiveStates[key].pendingMessages); got != 2 {
		t.Fatalf("pending=%d, want 2", got)
	}
	for _, q := range e.interactiveStates[key].pendingMessages {
		if q.messageID == "run_b" {
			t.Fatal("wrong entry removed")
		}
	}
	if e.PurgeQueuedMessage("run_missing", "u1", "chan") {
		t.Fatal("purge of unknown id must return false")
	}
	if e.PurgeQueuedMessage("run_a", "u2", "chan") {
		t.Fatal("purge with mismatched user must return false")
	}
	if e.PurgeQueuedMessage("run_a", "u1", "other") {
		t.Fatal("purge with mismatched channel must return false")
	}
}
