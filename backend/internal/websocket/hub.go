package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/oaknore/pms3/internal/middleware"
)

// WSMessage is what the server pushes to connected clients.
type WSMessage struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	CreatedAt time.Time   `json:"created_at"`
}

// client represents one connected browser tab / mobile connection.
type client struct {
	userID uuid.UUID
	orgID  uuid.UUID
	conn   *websocket.Conn
	send   chan []byte
}

// Hub manages all active WebSocket connections.
type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID][]*client // userID → connections (multi-tab)

	broadcast  chan *broadcastMsg
	register   chan *client
	unregister chan *client

	pingInterval time.Duration
	pongWait     time.Duration
	writeWait    time.Duration
}

type broadcastMsg struct {
	recipientID uuid.UUID // zero = org-wide
	orgID       uuid.UUID
	msg         []byte
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS for WS is handled at the HTTP level
	},
}

func NewHub(pingInterval, pongWait, writeWait time.Duration) *Hub {
	return &Hub{
		clients:      make(map[uuid.UUID][]*client),
		broadcast:    make(chan *broadcastMsg, 256),
		register:     make(chan *client, 32),
		unregister:   make(chan *client, 32),
		pingInterval: pingInterval,
		pongWait:     pongWait,
		writeWait:    writeWait,
	}
}

// Run starts the hub event loop — call in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c.userID] = append(h.clients[c.userID], c)
			h.mu.Unlock()
			log.Printf("ws: user %s connected", c.userID)

		case c := <-h.unregister:
			h.mu.Lock()
			list := h.clients[c.userID]
			for i, cc := range list {
				if cc == c {
					h.clients[c.userID] = append(list[:i], list[i+1:]...)
					close(c.send)
					break
				}
			}
			if len(h.clients[c.userID]) == 0 {
				delete(h.clients, c.userID)
			}
			h.mu.Unlock()
			log.Printf("ws: user %s disconnected", c.userID)

		case bcast := <-h.broadcast:
			h.mu.RLock()
			if bcast.recipientID != (uuid.UUID{}) {
				// targeted delivery
				for _, c := range h.clients[bcast.recipientID] {
					select {
					case c.send <- bcast.msg:
					default:
						// slow consumer — drop
					}
				}
			} else {
				// org-wide broadcast
				for uid, list := range h.clients {
					_ = uid
					for _, c := range list {
						if c.orgID == bcast.orgID {
							select {
							case c.send <- bcast.msg:
							default:
							}
						}
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SendToUser delivers a typed message to a specific user (all their tabs).
func (h *Hub) SendToUser(userID uuid.UUID, msgType string, payload interface{}) {
	data, err := json.Marshal(WSMessage{Type: msgType, Payload: payload, CreatedAt: time.Now()})
	if err != nil {
		return
	}
	h.broadcast <- &broadcastMsg{recipientID: userID, msg: data}
}

// SendToOrg broadcasts to every connected user in an org.
func (h *Hub) SendToOrg(orgID uuid.UUID, msgType string, payload interface{}) {
	data, err := json.Marshal(WSMessage{Type: msgType, Payload: payload, CreatedAt: time.Now()})
	if err != nil {
		return
	}
	h.broadcast <- &broadcastMsg{orgID: orgID, msg: data}
}

// ServeWS upgrades an HTTP connection to WebSocket and registers the client.
// Requires the Authenticate middleware to have run first.
func (h *Hub) ServeWS(jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.UserIDFrom(r.Context())
		orgID := middleware.OrgIDFrom(r.Context())
		if userID == (uuid.UUID{}) {
			log.Printf("ws: unauthorized - missing userID")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws upgrade error: %v", err)
			return
		}

		log.Printf("ws: connection established for user %s", userID)
		c := &client{
			userID: userID,
			orgID:  orgID,
			conn:   conn,
			send:   make(chan []byte, 64),
		}
		h.register <- c

		go h.writePump(c)
		go h.readPump(c)
	}
}

func (h *Hub) writePump(c *client) {
	ticker := time.NewTicker(h.pingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(h.writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(h.writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) readPump(c *client) {
	defer func() {
		h.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(h.pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(h.pongWait))
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
