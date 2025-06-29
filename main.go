package main

import (
	"fmt"
    "os"
	"net/http"
    "encoding/json"
    "database/sql"

	"github.com/gorilla/websocket"
    "github.com/joho/godotenv"
    _ "github.com/lib/pq"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections for now
	},
}

type client struct {
    conn *websocket.Conn
    send chan []byte
    username string
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

        var incoming map[string]string
        if err := json.Unmarshal(msg, &incoming); err != nil {
            continue
        }

        switch incoming["type"] {
        case "join":
            c.username = incoming["username"]

            // Send last 50 messages to just this user
            rows, err := db.Query(`SELECT username, message FROM messages ORDER BY id DESC LIMIT 50`)
            if err == nil {
                defer rows.Close()
                var history []string
                for rows.Next() {
                    var uname, m string
                    rows.Scan(&uname, &m)
                    history = append([]string{fmt.Sprintf("%s: %s", uname, m)}, history...)
                }
                for _, h := range history {
                    c.send <- []byte(h)
                }
            }

            // Notify everyone that this user joined
            joinMsg := fmt.Sprintf("%s joined the chat", c.username)
            chatHub.broadcast <- broadcastMsg{sender: nil, message: []byte(joinMsg)}

        case "chat":
            text := fmt.Sprintf("%s: %s", c.username, incoming["message"])

            // Save message in DB
            _, err := db.Exec(`INSERT INTO messages (username, message) VALUES ($1, $2)`, c.username, incoming["message"])
            if err != nil {
                fmt.Println("DB insert error:", err)
            }

            // Broadcast message
            chatHub.broadcast <- broadcastMsg{sender: c, message: []byte(text)}
        }
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

var db *sql.DB

func initDB() {
    godotenv.Load()
    connStr := os.Getenv("DATABASE_URL")

    var err error
    db, err = sql.Open("postgres", connStr)
    if err != nil {
        panic(err)
    }

    if err := db.Ping(); err != nil {
        panic(err)
    }

    fmt.Println("Connected to database")
}

func main() {
    initDB()
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
