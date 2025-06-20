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
	defer conn.Close()

	for {
		// Read message from browser
		_, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Read error: ", err)
			break
		}
		fmt.Printf("Recieved: %s\n", msg)

		// Echo message back
		err = conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			fmt.Println("Write error: ", err)
			break
		}
	}
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/ws", wsHandler)

	fmt.Println("Server is running on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
