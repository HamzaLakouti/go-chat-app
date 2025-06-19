package main

import (
	"fmt"
	"net/http"
)

// Home page handler
func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the Chat App!")
}

// Chat page handler
func chatHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "chat.html")
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/chat", chatHandler)

	fmt.Println("Server is running on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
