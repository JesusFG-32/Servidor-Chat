package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

type User struct {
	ID             string
	Username       string
	Email          string
	PasswordHash   string
	CreatedAt      time.Time
	LastConnection time.Time
}

func ConnectDB() {
	// Try to get credentials from environment variables, fallback to defaults
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "chat_user"
	}

	pass := os.Getenv("DB_PASS")
	if pass == "" {
		pass = "chat_pass"
	}

	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "127.0.1.1:3306" // Typical on local linux
	}

	dbname := "chat_app"

	// DSN: user:password@tcp(host:port)/dbname?parseTime=true
	var dsn string
	if pass == "" {
		dsn = fmt.Sprintf("%s@tcp(%s)/%s?parseTime=true", user, host, dbname)
	} else {
		dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, pass, host, dbname)
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("Failed to open DB connection: %v\n", err)
		return
	}

	// Wait for DB to be available up to 5 seconds
	for i := 0; i < 5; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		log.Printf("Failed to ping DB, it might be down: %v\n", err)
	} else {
		log.Println("Database connection established!")
	}

	DB = db
}
