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

	http.HandleFunc("/app/chat/api/register", RegisterHandler)
	http.HandleFunc("/app/chat/api/login", LoginHandler)
	http.HandleFunc("/app/chat/api/logout", LogoutHandler)
	http.HandleFunc("/app/chat/api/session", SessionHandler)

	http.HandleFunc("/api/register", RegisterHandler)
	http.HandleFunc("/api/login", LoginHandler)
	http.HandleFunc("/api/logout", LogoutHandler)
	http.HandleFunc("/api/session", SessionHandler)

	http.HandleFunc("/register", RegisterHandler)
	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/logout", LogoutHandler)
	http.HandleFunc("/session", SessionHandler)

	http.HandleFunc("/app/chat/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	})

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path
		basePath := "/"

		if len(urlPath) >= 4 && urlPath[:4] == "/api" || len(urlPath) >= 13 && urlPath[:13] == "/app/chat/api" {
			http.Error(w, "API Route Not Found: "+urlPath, http.StatusNotFound)
			return
		}

		if len(urlPath) >= 9 && urlPath[:9] == "/app/chat" {
			basePath = "/app/chat"
			urlPath = urlPath[9:]
			if urlPath == "" {
				urlPath = "/"
			}
		}

		if urlPath == "/ws" || urlPath == "/ws/" {
			ServeWs(hub, w, r)
			return
		}

		isValidSession, _ := ValidateSession(r)
		/*
			if urlPath == "/room" && !isValidSession {
				redirectTarget := basePath
				if redirectTarget == "/app/chat" {
					redirectTarget += "/"
				}
				log.Println("Redirecting user to " + redirectTarget + " from /room (l1)")

				log.Println("Redirecting user to /app/chat/room from /room (l2)")
				return
			}
		*/

		if urlPath == "/" && isValidSession {
			redirectTarget := basePath
			if redirectTarget == "/" {
				redirectTarget = "/room"
			} else {
				redirectTarget = basePath + "/room"
			}
			log.Println("Redirecting user to " + redirectTarget + " from / (l3)")
			http.Redirect(w, r, redirectTarget, http.StatusFound)
			return
		}

		path := "./public" + urlPath
		if _, err := os.Stat(path); os.IsNotExist(err) && urlPath != "/" {
			http.ServeFile(w, r, "./public/index.html")
			return
		}

		r.URL.Path = urlPath
		fs.ServeHTTP(w, r)
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
