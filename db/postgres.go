package db

import (
	"database/sql"
	"log"
	_ "github.com/lib/pq"
)
func Connect(postgresURL string) *sql.DB {
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging database: ", err)
	}

	log.Println("Connected to PostgreSQL")
	return db
}