package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func generateDeviceID() string {

	b := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		fmt.Sprintf("Failed to generate a random device ID: %x ERROR: %s", b, err)
	}

	return fmt.Sprintf("%x", b)

}

func OnboarDevice(w http.ResponseWriter, r *http.Request) {

	// To do is to create the logic behind getting the tenant associated with the user logged in
	const tenantID = "9052ef58-b79e-4684-a026-f39fd6f8f717"
	deviceID := ""
	flag := true

	// Generate device ID and Check if duplicate
	for flag {
		deviceID = generateDeviceID()
		dbName := fmt.Sprintf("Performance_%s", tenantID)
		// Check if deviceID exists in the database
		query := fmt.Sprintf("SELECT device_id FROM `%s`.Devices WHERE device_id = '%s'", dbName, deviceID)
		rows, err := db.Query(query)

		if err != nil {
			http.Error(w, fmt.Sprintf("Error querying devices: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		if !rows.Next() {
			flag = false
		}
	}

	// Now write onboarding script

	scriptContent := fmt.Sprintf(`#!/bin/bash
	# Define variables
	tenantID=%s
	deviceID=%s

	# Ensure the target directory exists
	mkdir -p /opt/cloud-vigilante

	# Generate the JSON object and store it
	echo "{\"TenantID\": \"$tenantID\", \"DeviceID\": \"$deviceID\"}" > /opt/cloud-vigilante/cloudVigilanteOnboarding.json

	echo "JSON object stored successfully."`, tenantID, deviceID)

	// Write the script to a file
	wd, err := os.Getwd()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting working directory: %v", err), http.StatusInternalServerError)
		fmt.Printf("Error getting working directory: %v", err)
		return
	}

	scriptPath := fmt.Sprintf("%s/downloads/onboard_%s.sh", wd, deviceID)

	file, err := os.Create(scriptPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating script file: %v", err), http.StatusInternalServerError)
		fmt.Printf("Error creating script file: %v", err)
		return
	}

	// Close handle to file
	defer file.Close()

	// Write the script content to the file
	_, err = file.WriteString(scriptContent)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error writing to script file: %v", err), http.StatusInternalServerError)
		fmt.Printf("Error writing to script file: %v", err)
		return
	}

	fmt.Sprint("Onboard script created successfully: %s", scriptPath)

	// Return link to download the script

	type DownloadLink struct {
		Link string `json:"downloadLink"`
	}

	link := fmt.Sprintf("http://cloudvigilante.anoliveira.com/downloads/onboard_%s.sh", deviceID)

	// Set the response header to application/json
	w.Header().Set("Content-Type", "application/json")

	// Create an instance of DownloadLink with the actual download link
	downloadLink := DownloadLink{Link: link}

	// Encode the downloadLink instance as JSON into the response
	if err := json.NewEncoder(w).Encode(downloadLink); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		return
	}

}
