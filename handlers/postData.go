package handlers

import (
	"cloudVigilante/backend/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Define the Go structs that match the JSON structure
type TotalConsumption struct {
	TotalCPU       float64 `json:"TotalCpu"`
	TotalMemory    int64   `json:"TotalMemory"`
	UsedMemory     int64   `json:"UsedMemory"`
	UsedMemoryPerc float64 `json:"UsedMemoryP"`
}

type MachineProperties struct {
	DeviceID   string `json:"deviceID"`
	TenantID   string `json:"tenantID"`
	DeviceName string `json:"deviceName"`
	MacAddress string `json:"macAddress"`
	IPAddress  string `json:"ipAddress"`
	TimeStamp  string `json:"timeStamp"`
}

type ProcessInfo struct {
	ProcessPID      int     `json:"processPID"`
	ProcessName     string  `json:"processName"`
	ProcessCommand  string  `json:"processCommand"`
	ProcessCpuUsage float64 `json:"ProcessCpuUsage"`
	ProcessMemUsage int64   `json:"ProcessMemUsage"`
}

type PerformanceData struct {
	TotalConsumption  TotalConsumption  `json:"totalConsumption"`
	MachineProperties MachineProperties `json:"machineProperties"`
	ProcessInfo       []ProcessInfo     `json:"processInfo"`
}

// Declare global db var
var db *sql.DB

// Function to set the database connection
func SetDB(database *sql.DB) {
	db = database
}

func ReceivePerformanceMetrics(w http.ResponseWriter, r *http.Request) {

	var performanceData PerformanceData

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Print body out for debugging
	// fmt.Println(string(body))

	// Parse the JSON data
	if err := json.Unmarshal(body, &performanceData); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		fmt.Println(err)
		return
	}

	orgID := performanceData.MachineProperties.TenantID

	// Ensure the PerformanceDB for the organization exists
	err = models.CreatePerformanceDB(db, orgID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating PerformanceDB: %v", err), http.StatusInternalServerError)
		return
	}

	deviceData := models.DeviceData{
		DeviceID:   performanceData.MachineProperties.DeviceID,
		Hostname:   performanceData.MachineProperties.DeviceName,
		MACAddress: performanceData.MachineProperties.MacAddress,
		IPAddress:  performanceData.MachineProperties.IPAddress,
	}

	performance := models.PerformanceData{
		DeviceID:    performanceData.MachineProperties.DeviceID,
		Timestamp:   performanceData.MachineProperties.TimeStamp,
		CPUUsage:    performanceData.TotalConsumption.TotalCPU,
		RAMUsage:    performanceData.TotalConsumption.UsedMemory,
		TotalMemory: performanceData.TotalConsumption.TotalMemory,
		UsedMemoryP: performanceData.TotalConsumption.UsedMemoryPerc,
		Processes:   make([]models.ProcessData, len(performanceData.ProcessInfo)),
	}

	for i, process := range performanceData.ProcessInfo {
		performance.Processes[i] = models.ProcessData{
			PID:      process.ProcessPID,
			Name:     process.ProcessName,
			Command:  process.ProcessCommand,
			CPUUsage: process.ProcessCpuUsage,
			RAMUsage: process.ProcessMemUsage,
		}
	}

	err = models.InsertPerformanceData(db, orgID, deviceData, performance)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error inserting performance data: %v", err), http.StatusInternalServerError)
		return
	}

	// Respond to the client
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Performance data recorded successfully"))

}
