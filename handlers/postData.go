package handlers

import (
	"fmt"
	"io"
	"net/http"
)

func ReceivePerformanceMetrics(w http.ResponseWriter, r *http.Request) {

	// Get the request and print it out
	body, _ := io.ReadAll(r.Body)

	fmt.Println("Received POST request:")
	fmt.Println(string(body))

	// Respond to the client
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Request Received"))

}
