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

	// Handle CORS
	mux := http.NewServeMux()

	// Handle POST routes
	http.HandleFunc("/api/v1/postMetrics", handlers.ReceivePerformanceMetrics)
	mux.Handle("/api/v1/cpumetrics", handlers.EnableCORS(http.HandlerFunc(handlers.RetrieveCPUMetrics)))

	// Handle GET routes

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", mux))

}
