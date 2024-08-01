package main

import (
	"database/sql"
	"fmt"
	_ "fmt"
	_ "github.com/mattn/go-sqlite3"
	_ "golang.org/x/crypto/bcrypt"
)

var db *sql.DB

func initDB() error {
	fmt.Println("Initializing database...")
	var err error
	db, err = sql.Open("sqlite3", "./Database/urlshortener.db")
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	fmt.Println("Database opened successfully")

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT UNIQUE NOT NULL,
            password TEXT NOT NULL
        )
    `)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS urls (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            user_id INTEGER,
            short_code TEXT UNIQUE NOT NULL,
            long_url TEXT NOT NULL,
            custom_name TEXT,
            expires_at DATETIME,
            clicks INTEGER DEFAULT 0,
            FOREIGN KEY (user_id) REFERENCES users(id)
        )
    `)
	fmt.Println("Database initialization complete")
	return err
}
