package main

import (
	"cloudVigilante/backend/handlers"
	"log"
	"net/http"
)

func main() {

	http.HandleFunc("/api/v1/postMetrics", handlers.ReceivePerformanceMetrics)

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
