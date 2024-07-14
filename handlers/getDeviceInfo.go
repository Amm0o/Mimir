package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type DeviceInfoResponse struct {
	DeviceId   string `json:"device_id"`
	DeviceName string `json:"device_hostname"`
	MacAddress string `json:"mac_address"`
	IpAddress  string `json:"ip_address"`
}

// Function to query Devices table for a tenant and return device information
// URL needs to contain the tenantID ID in the url as a query parameter with the following format:
// /getDeviceInfo?tenantID=1234
func GetDeviceInfo(w http.ResponseWriter, r *http.Request) {

	fmt.Println("GetDeviceInfo called")

	var devices []DeviceInfoResponse

	// Extract tenantID from URL
	tenantID := r.URL.Query().Get("tenantID")
	if tenantID == "" {
		http.Error(w, "Invalid tenantID", http.StatusBadRequest)
		fmt.Println("Invalid tenantID, %s", tenantID)
	}

	// Query the Devices table for the tenantID
	dbName := fmt.Sprintf("Performance_%s", tenantID)

	// construct the query
	query := fmt.Sprintf(`
		SELECT device_id, device_hostname, mac_address, ip_address from %s.Devices
	`, fmt.Sprintf("`%s`", dbName))

	// Execute the query
	rows, err := db.Query(query)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error querying device information: %v", err), http.StatusInternalServerError)
		fmt.Printf("Error querying device information: %v", err)
		return
	}

	fmt.Println("Query executed successfully, device:")

	// Iterate over the rows and populate the devices slice
	for rows.Next() {
		var device DeviceInfoResponse

		if err := rows.Scan(&device.DeviceId, &device.DeviceName, &device.MacAddress, &device.IpAddress); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning device information: %v", err), http.StatusInternalServerError)
			fmt.Printf("Error scanning device information: %v", err)
			return
		}

		// Append the device into the devices slice
		devices = append(devices, device)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Error iterating over device information rows: %v", err), http.StatusInternalServerError)
		fmt.Printf("Error iterating over device information rows: %v", err)
		return
	}

	// Close handle to DB
	defer rows.Close()

	// return the devices slice as a JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)

}
