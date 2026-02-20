package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	// Initialize database connection
	ConnectDB()

	// Initialize WebSocket Hub
	hub := NewHub()
	go hub.Run()

	// Create directories for static files if they don't exist
	os.MkdirAll("public", os.ModePerm)

	// Static and SPA routes
	fs := http.FileServer(http.Dir("./public"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := "./public" + r.URL.Path
		if _, err := os.Stat(path); os.IsNotExist(err) && r.URL.Path != "/" {
			// Fallback for SPA Routing (e.g. /chat)
			http.ServeFile(w, r, "./public/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})

	// API Auth Routes
	http.HandleFunc("/api/register", RegisterHandler)
	http.HandleFunc("/api/login", LoginHandler)
	http.HandleFunc("/api/logout", LogoutHandler)
	http.HandleFunc("/api/session", SessionHandler)

	// WebSocket Route
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
