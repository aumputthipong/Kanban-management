package websocket

import (
	"github.com/aumputthipong/mini-erp-kanban/backend/internal/service"
)

// Hub manages client connections grouped into one room per board ID. It
// delegates write operations to BoardCommandService rather than holding
// *db.Queries directly.
type Hub struct {
	rooms map[string]map[*Client]bool

	broadcast  chan BroadcastMessage
	register   chan *Client
	unregister chan *Client
	stop       chan struct{}
	boardCmd   *service.BoardCommandService
	activities *service.ActivityService
	// allowedOrigin is the single trusted browser origin (FRONTEND_URL).
	// Empty string disables origin checking — only acceptable in tests.
	allowedOrigin string
}

type BroadcastMessage struct {
	BoardID string
	Message []byte
}

func NewHub(boardCmd *service.BoardCommandService, activities *service.ActivityService, allowedOrigin string) *Hub {
	return &Hub{
		rooms:         make(map[string]map[*Client]bool),
		broadcast:     make(chan BroadcastMessage),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		stop:          make(chan struct{}),
		boardCmd:      boardCmd,
		activities:    activities,
		allowedOrigin: allowedOrigin,
	}
}

// Shutdown closes every active WS connection and stops the hub goroutine.
// Idempotent — safe to call once at SIGTERM. Pumps observe a closed `send`
// channel and exit; ReadPump returns when the underlying conn closes.
func (h *Hub) Shutdown() {
	select {
	case <-h.stop:
		// already stopped
	default:
		close(h.stop)
	}
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.stop:
			for boardID, clients := range h.rooms {
				for client := range clients {
					close(client.send)
					_ = client.conn.Close()
				}
				delete(h.rooms, boardID)
			}
			return
		case client := <-h.register:
			if _, ok := h.rooms[client.boardID]; !ok {
				h.rooms[client.boardID] = make(map[*Client]bool)
			}
			h.rooms[client.boardID][client] = true

		case client := <-h.unregister:
			if clients, ok := h.rooms[client.boardID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)

					// Drop the room once empty to free memory.
					if len(clients) == 0 {
						delete(h.rooms, client.boardID)
					}
				}
			}

		case broadcastMsg := <-h.broadcast:
			if clients, ok := h.rooms[broadcastMsg.BoardID]; ok {
				for client := range clients {
					select {
					case client.send <- broadcastMsg.Message:
					default:
						// Slow or dead client: drop it so a full send buffer
						// can't block the hub.
						close(client.send)
						delete(clients, client)
					}
				}
			}
		}
	}
}
