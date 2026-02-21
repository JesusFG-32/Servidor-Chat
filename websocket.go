package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all cross-origin for dev
	},
}

type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	Username string
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
}

type WSMessage struct {
	Type     string   `json:"type"`               // "chat" or "users"
	Username string   `json:"username,omitempty"` // For chat messages
	Content  string   `json:"content,omitempty"`  // For chat messages
	Users    []string `json:"users,omitempty"`    // For the connected users list
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) BroadcastActiveUsers() {
	var activeUsers []string
	for client := range h.clients {
		activeUsers = append(activeUsers, client.Username)
	}

	msg := WSMessage{
		Type:  "users",
		Users: activeUsers,
	}
	b, _ := json.Marshal(msg)

	for client := range h.clients {
		select {
		case client.send <- b:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.BroadcastActiveUsers()
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.BroadcastActiveUsers()
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(15 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			log.Printf("readPump error or closure for user %s: %v", c.Username, err)
			break
		}

		if string(message) == `{"type":"ping"}` {
			c.conn.SetReadDeadline(time.Now().Add(15 * time.Second))
			continue
		}

		msg := WSMessage{
			Type:     "chat",
			Username: c.Username,
			Content:  string(message),
		}
		b, _ := json.Marshal(msg)
		c.hub.broadcast <- b
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(5 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				log.Printf("writePump closing: send channel closed for user %s", c.Username)
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Printf("writePump closing: NextWriter error for user %s: %v", c.Username, err)
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				log.Printf("writePump closing: writer close error for user %s: %v", c.Username, err)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("writePump closing: PingMessage fail for user %s: %v", c.Username, err)
				return
			}
		}
	}
}

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	tokenString := r.URL.Query().Get("token")
	var validToken *jwt.Token
	var claims *Claims

	// Try query parameter first
	if tokenString != "" {
		c := &Claims{}
		t, err := jwt.ParseWithClaims(tokenString, c, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		log.Printf("WS Auth Debug [Query] - Token: %s, Parse Error: %v", tokenString, err)
		if err == nil && t.Valid {
			validToken = t
			claims = c
		}
	}

	if validToken == nil {
		for _, cookie := range r.Cookies() {
			if cookie.Name == "token" {
				c := &Claims{}
				t, err := jwt.ParseWithClaims(cookie.Value, c, func(token *jwt.Token) (interface{}, error) {
					return jwtKey, nil
				})
				log.Printf("WS Auth Debug [Cookie] - Parse Error: %v", err)
				if err == nil && t.Valid {
					validToken = t
					claims = c
					break
				}
			}
		}
	}

	if validToken == nil {
		log.Println("WS Auth Debug - No valid token found, rejecting connection")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256), Username: claims.Username}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
