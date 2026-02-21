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

	var dsnNoDB string
	if pass == "" {
		dsnNoDB = fmt.Sprintf("%s@tcp(%s)/?parseTime=true&multiStatements=true", user, host)
	} else {
		dsnNoDB = fmt.Sprintf("%s:%s@tcp(%s)/?parseTime=true&multiStatements=true", user, pass, host)
	}

	dbInit, err := sql.Open("mysql", dsnNoDB)
	if err != nil {
		log.Printf("Failed to open initial DB connection: %v\n", err)
		return
	}

	for i := 0; i < 5; i++ {
		err = dbInit.Ping()
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		log.Printf("Failed to ping DB server, it might be down: %v\n", err)
	} else {
		schemaBytes, schemaErr := os.ReadFile("schema.sql")
		if schemaErr == nil {
			_, err = dbInit.Exec(string(schemaBytes))
			if err != nil {
				log.Printf("Failed to execute schema.sql: %v\n", err)
			} else {
				log.Println("Database and tables checked/created successfully!")
			}
		} else {
			log.Printf("Could not read schema.sql: %v\n", schemaErr)
			// Fallback: just create the database
			_, err = dbInit.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s;", dbname))
			if err != nil {
				log.Printf("Failed to create database: %v\n", err)
			}
		}
	}
	dbInit.Close()

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

	err = db.Ping()
	if err != nil {
		log.Printf("Failed to connect to specific database: %v\n", err)
	} else {
		log.Println("Database connection established!")
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id CHAR(36) PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_connection TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Printf("Failed to ensure users table exists: %v\n", err)
	} else {
		log.Println("Users table checked/created successfully.")
	}

	DB = db
}
