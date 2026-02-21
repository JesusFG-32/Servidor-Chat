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
	user := os.Getenv("DB_USER")
	if user == "" {
		log.Printf("La variable de entorno DB_USER no está establecida.\n")
		log.Printf("Establece la variable de entorno DB_USER con el usuario de la base de datos.\n")
		log.Printf("Saliendo dentro de 6 segundos...")
		time.Sleep(6 * time.Second)
		os.Exit(-2)
	}

	pass := os.Getenv("DB_PASS")
	if pass == "" {
		log.Printf("La variable de entorno DB_PASS no está establecida.\n")
		log.Printf("Establece la variable de entorno DB_PASS con la contraseña de la base de datos.\n")
		log.Printf("Saliendo dentro de 6 segundos...")
		time.Sleep(6 * time.Second)
		os.Exit(-2)
	}

	host := os.Getenv("DB_HOST")
	if host == "" {
		log.Printf("La variable de entorno DB_HOST no está establecida.\n")
		log.Printf("Establece la variable de entorno DB_HOST con el host de la base de datos.\n")
		log.Printf("Saliendo dentro de 6 segundos...")
		time.Sleep(6 * time.Second)
		os.Exit(-2)
	}

	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		log.Printf("La variable de entorno DB_NAME no está establecida.\n")
		log.Printf("Establece la variable de entorno DB_NAME con el nombre de la base de datos.\n")
		log.Printf("Saliendo dentro de 6 segundos...")
		time.Sleep(6 * time.Second)
		os.Exit(-2)
	}

	var dsnNoDB string
	if pass == "" {
		dsnNoDB = fmt.Sprintf("%s@tcp(%s)/?parseTime=true&multiStatements=true", user, host)
	} else {
		dsnNoDB = fmt.Sprintf("%s:%s@tcp(%s)/?parseTime=true&multiStatements=true", user, pass, host)
	}

	dbInit, err := sql.Open("mysql", dsnNoDB)
	if err != nil {
		log.Printf("Algo fallo al abrir la conexión con la base de datos: %v\n", err)
		log.Printf("Saliendo dentro de 6 segundos...")
		time.Sleep(6 * time.Second)
		os.Exit(-2)
	}

	for i := 0; i < 5; i++ {
		err = dbInit.Ping()
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		log.Printf("No se pudo contactar al servidor de la base de datos, podría estar caído: %v\n", err)
	} else {
		schemaQuery := fmt.Sprintf(`
		CREATE DATABASE IF NOT EXISTS %s;
		USE %s;
		CREATE TABLE IF NOT EXISTS users (
			id CHAR(36) PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_connection TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		);`, dbname, dbname)

		_, err = dbInit.Exec(schemaQuery)
		if err != nil {
			log.Printf("Algo fallo al inicializar la base de datos y tablas internas: %v\n", err)
		} else {
			log.Println("Base de datos y tablas configuradas con éxito.")
		}

		hostname := "localhost"
		if len(host) > 0 {
			for i := 0; i < len(host); i++ {
				if host[i] == ':' {
					hostname = host[:i]
					break
				}
			}
			if hostname == host {
				hostname = host
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
		log.Printf("Algo fallo al abrir la conexión con la base de datos: %v\n", err)
		return
	}

	err = db.Ping()
	if err != nil {
		log.Printf("No se ha podido conectar con la base de datos especificada: %v\n", err)
	} else {
		log.Println("Conexión con la base de datos establecida!")
	}
	DB = db
}
