package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("config.env")
	if err != nil {
		log.Println("No config.env file found, using default environment variables")
	}
	ConnectDB()
	hub := NewHub()
	go hub.Run()

	os.MkdirAll("public", os.ModePerm)

	fs := http.FileServer(http.Dir("./public"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := "./public" + r.URL.Path
		if _, err := os.Stat(path); os.IsNotExist(err) && r.URL.Path != "/" {
			http.ServeFile(w, r, "./public/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})

	http.HandleFunc("/api/register", RegisterHandler)
	http.HandleFunc("/api/login", LoginHandler)
	http.HandleFunc("/api/logout", LogoutHandler)
	http.HandleFunc("/api/session", SessionHandler)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on port %s", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
