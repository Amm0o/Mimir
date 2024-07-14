package main

import (
	"cloudVigilante/backend/handlers"
	"cloudVigilante/backend/models"
	"fmt"
	"log"
	"net/http"
	"os"

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

	// Start the file server
	wd, err := os.Getwd()

	if err != nil {
		fmt.Printf("Error getting working directory for the file server: %v", err)
		return
	}

	directory := fmt.Sprintf("%s/downloads", wd)
	fileServer := http.FileServer(http.Dir(directory))
	mux.Handle("/downloads/", http.StripPrefix("/downloads/", fileServer))

	log.Printf("Started serving files from %s", directory)

	// Handle POST routes
	mux.Handle("/api/v1/postmetrics", handlers.EnableCORS(http.HandlerFunc(handlers.ReceivePerformanceMetrics)))
	mux.Handle("/api/v1/cpumetrics", handlers.EnableCORS(http.HandlerFunc(handlers.RetrieveCPUMetrics)))
	mux.Handle("/api/v1/rammetrics", handlers.EnableCORS(http.HandlerFunc(handlers.RetrieveRamMetrics)))

	// Handle GET routes
	mux.Handle("/api/v1/getdeviceinfo", handlers.EnableCORS((http.HandlerFunc(handlers.GetDeviceInfo))))
	mux.Handle("/api/v1/onboard-device", handlers.EnableCORS((http.HandlerFunc(handlers.OnboarDevice))))

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", mux))

}
