package models

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

func ConnectToDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", "cloudvigilante:cloudvigilante@tcp(127.0.0.1:3306)/")
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("Conncted to db server :)")

	// Verify the connection is valid
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging database: %w", err)
	}
	fmt.Println("Pinged DB :)")

	return db, nil
}
