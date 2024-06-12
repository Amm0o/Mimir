package main

import (
	"cloudVigilante/backend/handlers"
	"cloudVigilante/backend/models"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
)

func main() {

	db, err = models.ConnectToDB()

	if err != nil {
		fmt.Println("Error connecting to db", err)
		return
	}

	defer db.Close()

	handlers.SetDB(db)

	// Handle routes
	http.HandleFunc("/api/v1/postMetrics", handlers.ReceivePerformanceMetrics)

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}
