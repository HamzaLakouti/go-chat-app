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

type client struct {
    conn *websocket.Conn
    send chan []byte
}

// Wrap message with its sender
type broadcastMsg struct {
    sender  *client
    message []byte
}

type hub struct {
    clients    map[*client]bool
    broadcast  chan broadcastMsg
    register   chan *client
    unregister chan *client
}

var chatHub = hub{
    clients:    make(map[*client]bool),
    broadcast:  make(chan broadcastMsg),
    register:   make(chan *client),
    unregister: make(chan *client),
}

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
        case bm := <-h.broadcast:
            for client := range h.clients {
                if client == bm.sender {
                    // skip the origin
                    continue
                }
                select {
                case client.send <- bm.message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Println("Upgrade error:", err)
        return
    }
    c := &client{conn: conn, send: make(chan []byte)}
    chatHub.register <- c

    go c.writePump()
    c.readPump()
}

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
        // include the sender in the broadcast
        chatHub.broadcast <- broadcastMsg{sender: c, message: msg}
    }
}

func (c *client) writePump() {
    defer c.conn.Close()
    for msg := range c.send {
        if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
            break
        }
    }
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "Welcome to the Chat App!")
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "chat.html")
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
