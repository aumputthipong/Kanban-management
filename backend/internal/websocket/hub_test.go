package websocket

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Cleanup unregisters clients before Shutdown, which would close their nil conns.
type testHub struct {
	*Hub
	clients []*Client
}

func startHub(t *testing.T) *testHub {
	t.Helper()
	th := &testHub{Hub: NewHub("")}
	go th.Run()
	t.Cleanup(func() {
		for _, c := range th.clients {
			th.unregister <- c
		}
		th.Shutdown()
	})
	return th
}

func joinRoom(h *testHub, boardID, userID string) *Client {
	c := &Client{hub: h.Hub, send: make(chan []byte, 8), boardID: boardID, userID: userID}
	h.register <- c
	h.clients = append(h.clients, c)
	return c
}

func receive(t *testing.T, c *Client) (msg string, ok bool) {
	t.Helper()
	select {
	case m, open := <-c.send:
		return string(m), open
	case <-time.After(time.Second):
		t.Fatal("timed out waiting on client channel")
		return "", false
	}
}

func TestEvictUser_ClosesOnlyThatUsersConnectionsOnThatBoard(t *testing.T) {
	h := startHub(t)
	removedTab1 := joinRoom(h, "board-1", "user-removed")
	removedTab2 := joinRoom(h, "board-1", "user-removed")
	stays := joinRoom(h, "board-1", "user-stays")
	otherBoard := joinRoom(h, "board-2", "user-removed")

	h.EvictUser("board-1", "user-removed")
	h.Broadcast("board-1", []byte("after"))
	h.Broadcast("board-2", []byte("other"))

	for _, c := range []*Client{removedTab1, removedTab2} {
		_, ok := receive(t, c)
		assert.False(t, ok, "every tab of the removed user must be disconnected")
	}
	msg, ok := receive(t, stays)
	require.True(t, ok)
	assert.Equal(t, "after", msg, "remaining members keep receiving")
	msg, ok = receive(t, otherBoard)
	require.True(t, ok)
	assert.Equal(t, "other", msg, "the user's rooms on other boards are untouched")
}

// The removed client still gets the last queued message.
func TestEvictUser_DeliversMessagesQueuedBeforeEviction(t *testing.T) {
	h := startHub(t)
	removed := joinRoom(h, "board-1", "user-removed")

	h.Broadcast("board-1", []byte("members-updated"))
	h.EvictUser("board-1", "user-removed")

	msg, ok := receive(t, removed)
	require.True(t, ok)
	assert.Equal(t, "members-updated", msg)
	_, ok = receive(t, removed)
	assert.False(t, ok)
}

// ReadPump's unregister must not double-close the channel.
func TestEvictUser_ThenUnregister_DoesNotPanic(t *testing.T) {
	h := startHub(t)
	removed := joinRoom(h, "board-1", "user-removed")

	h.EvictUser("board-1", "user-removed")
	h.unregister <- removed
	h.Broadcast("board-1", []byte("still running"))

	_, ok := receive(t, removed)
	assert.False(t, ok)
}

func TestEvictUser_UnknownRoom_IsNoop(t *testing.T) {
	h := startHub(t)
	c := joinRoom(h, "board-1", "user-a")

	h.EvictUser("board-404", "user-a")
	h.Broadcast("board-1", []byte("hello"))

	msg, ok := receive(t, c)
	require.True(t, ok)
	assert.Equal(t, "hello", msg)
}
