package main

import (
	"cloudVigilante/backend/handlers"
	"cloudVigilante/backend/models"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

func main() {

	db, err := models.ConnectToDB()

	if err != nil {
		fmt.Println("Error connecting to db", err)
		return
	}

	defer db.Close()

	handlers.SetDB(db)

	// Handle POST routes
	http.HandleFunc("/api/v1/postMetrics", handlers.ReceivePerformanceMetrics)

	// Handle GET routes
	http.HandleFunc("/api/v1/cpumetrics", handlers.RetrieveCPUMetrics)

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}
