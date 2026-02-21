package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("config.env")
	if err != nil {
		log.Println("No se encontró el archivo config.env.")
		log.Println("Se creará un archivo nuevo y el servidor se detendrá.")
		os.WriteFile("config.env", []byte("DB_USER=root\nDB_PASS=\nDB_HOST=[IP_ADDRESS]"), 0644)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}
	ConnectDB()
	hub := NewHub()
	go hub.Run()

	os.MkdirAll("public", os.ModePerm)

	fs := http.FileServer(http.Dir("./public"))

	staticFS := http.FileServer(http.Dir("./public/assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", staticFS))

	http.HandleFunc("/inicio", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./public/index.html")
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/inicio", http.StatusFound)
			return
		}

		path := "./public" + r.URL.Path
		if _, err := os.Stat(path); os.IsNotExist(err) && r.URL.Path != "/" {
			http.Redirect(w, r, "/inicio", http.StatusFound)
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

	log.Printf("Servidor corriendo en puerto %s", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
