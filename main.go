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
	mux.Handle("/api/v1/postMetrics", handlers.EnableCORS(http.HandlerFunc(handlers.ReceivePerformanceMetrics)))
	mux.Handle("/api/v1/cpumetrics", handlers.EnableCORS(http.HandlerFunc(handlers.RetrieveCPUMetrics)))
	mux.Handle("/api/v1/rammetrics", handlers.EnableCORS(http.HandlerFunc(handlers.RetrieveRamMetrics)))

	// Handle GET routes

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", mux))

}
