package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections for now
	},
}

// Represents a single ws connection
type client struct {
	conn *websocket.Conn
	send chan []byte // channel of messages to send
}

// Manages all clients and broadcasts messages
type hub struct {
	clients    map[*client]bool
    broadcast  chan []byte
    register   chan *client
    unregister chan *client
}

var chatHub = hub{
    clients:    make(map[*client]bool),
    broadcast:  make(chan []byte),
    register:   make(chan *client),
    unregister: make(chan *client),
}

// Start the hub
func (h *hub) run() {
    for {
        select {
        case c := <-h.register:
            h.clients[c] = true
        case c := <-h.unregister:
            if _, ok := h.clients[c]; ok {
                delete(h.clients, c)
                close(c.send)
            }
        case msg := <-h.broadcast:
            for c := range h.clients {
                select {
                case c.send <- msg:
                default:
                    close(c.send)
                    delete(h.clients, c)
                }
            }
        }
    }
}

// Home page handler
func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the Chat App!")
}

// Chat page handler
func chatHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "chat.html")
}

// WS handler
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Upgrade error: ", err)
		return
	}
	
	c := &client{
		conn: conn,
		send: make(chan []byte),
	}

	chatHub.register <- c

    go c.writePump()
    c.readPump()
}

// Read messages from this client and broadcast
func (c *client) readPump() {
    defer func() {
        chatHub.unregister <- c
        c.conn.Close()
    }()

    for {
        _, msg, err := c.conn.ReadMessage()
        if err != nil {
            break
        }
        chatHub.broadcast <- msg
    }
}

// Write messages to this client
func (c *client) writePump() {
    defer c.conn.Close()
    for msg := range c.send {
        err := c.conn.WriteMessage(websocket.TextMessage, msg)
        if err != nil {
            break
        }
    }
}

func main() {
	go chatHub.run()
	
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/ws", wsHandler)

	fmt.Println("Server is running on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
