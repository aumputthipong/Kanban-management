package websocket

// Fan-out only: REST handlers persist, then Broadcast. The hub performs no writes.
type Hub struct {
	rooms map[string]map[*Client]bool

	broadcast  chan BroadcastMessage
	register   chan *Client
	unregister chan *Client
	evict      chan roomMember
	stop       chan struct{}
	// Empty disables origin checking — tests only.
	allowedOrigin string
}

type BroadcastMessage struct {
	BoardID string
	Message []byte
}

type roomMember struct {
	boardID string
	userID  string
}

func NewHub(allowedOrigin string) *Hub {
	return &Hub{
		rooms:         make(map[string]map[*Client]bool),
		broadcast:     make(chan BroadcastMessage),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		evict:         make(chan roomMember),
		stop:          make(chan struct{}),
		allowedOrigin: allowedOrigin,
	}
}

// No-op after shutdown, so late handlers never block.
func (h *Hub) Broadcast(boardID string, message []byte) {
	select {
	case h.broadcast <- BroadcastMessage{BoardID: boardID, Message: message}:
	case <-h.stop:
	}
}

// Membership is only checked at the handshake. Already-queued messages are still delivered.
func (h *Hub) EvictUser(boardID, userID string) {
	select {
	case h.evict <- roomMember{boardID: boardID, userID: userID}:
	case <-h.stop:
	}
}

// Idempotent.
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

					if len(clients) == 0 {
						delete(h.rooms, client.boardID)
					}
				}
			}

		case m := <-h.evict:
			clients := h.rooms[m.boardID]
			for client := range clients {
				if client.userID == m.userID {
					delete(clients, client)
					close(client.send)
				}
			}
			if clients != nil && len(clients) == 0 {
				delete(h.rooms, m.boardID)
			}

		case broadcastMsg := <-h.broadcast:
			if clients, ok := h.rooms[broadcastMsg.BoardID]; ok {
				for client := range clients {
					select {
					case client.send <- broadcastMsg.Message:
					default:
						// Drop slow clients so a full buffer can't block the hub.
						close(client.send)
						delete(clients, client)
					}
				}
			}
		}
	}
}
