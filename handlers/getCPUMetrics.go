package handlers

import (
	"cloudVigilante/backend/handlers/helpers"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
)

// Structs to match the JSON request
type TimeRange struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Metrics struct {
	CPULevel float64 `json:"cpuLevel"`
}

type Query struct {
	NumberOfProcesses int       `json:"numberOfProcesses"`
	Devices           []string  `json:"devices"`
	TimeRange         TimeRange `json:"timeRange"`
	Metrics           Metrics   `json:"metrics"`
}

type CPUMetricsRequest struct {
	TenantID string `json:"tenantID"`
	Query    Query  `json:"query"`
}

type CPUMetricsResponse struct {
	Timestamp       string  `json:"timestamp"`
	ProcessPID      int     `json:"processPID"`
	ProcessName     string  `json:"processName"`
	ProcessCommand  string  `json:"processCommand"`
	ProcessCPUUsage float64 `json:"processCpuUsage"`
}

type ProcessGroup struct {
	ProcessName string               `json:"processName"`
	ProcessPID  int                  `json:"processPID"`
	AvgCPU      float64              `json:"avgCpu"`
	Metrics     []CPUMetricsResponse `json:"metrics"`
}

type ProcessMetrics struct {
	ProcessName         string
	TotalCPUConsumption float64
	Metrics             map[int][]CPUMetricsResponse
}

type CpuProcessGroup struct {
	ProcessName string               `json:"processName"`
	AvgCpu      int64                `json:"avgCpu"`
	Metrics     []CPUMetricsResponse `json:"metrics"`
}

type CpuDeviceMetrics struct {
	DeviceID   string            `json:"DeviceID"`
	DeviceName string            `json:"DeviceName"`
	Metrics    []CpuProcessGroup `json:"Metrics"`
}

// Function to handle the retrieval of CPU metrics
func RetrieveCPUMetrics(w http.ResponseWriter, r *http.Request) {
	var cpuMetricsRequest CPUMetricsRequest

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse the JSON data
	if err := json.Unmarshal(body, &cpuMetricsRequest); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		fmt.Println(err)
		return
	}

	tenantID := cpuMetricsRequest.TenantID
	devices := cpuMetricsRequest.Query.Devices
	timeStart := cpuMetricsRequest.Query.TimeRange.Start
	timeEnd := cpuMetricsRequest.Query.TimeRange.End
	numberOfProcesses := cpuMetricsRequest.Query.NumberOfProcesses

	// Validate the tenant ID
	if tenantID == "" {
		http.Error(w, "Tenant ID is required", http.StatusBadRequest)
		return
	}

	// Determine the database name
	dbName := fmt.Sprintf("Performance_%s", tenantID)

	// Check if the tenant is registered
	var exists int
	query := fmt.Sprintf("SELECT EXISTS(SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = '%s')", dbName)
	err = db.QueryRow(query).Scan(&exists)

	if exists == 0 {
		log.Printf("DB: %s does not exist", dbName)
		http.Error(w, fmt.Sprintf("Error checking tenant: %s", tenantID), http.StatusInternalServerError)
		return
	}

	// If devices array is empty, query all devices
	deviceMap := make(map[string]string)
	if len(devices) == 0 {
		deviceQuery := fmt.Sprintf("SELECT device_id, device_hostname FROM `%s`.Devices", dbName)
		rows, err := db.Query(deviceQuery)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error querying devices: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var deviceID, deviceHostname string
			if err := rows.Scan(&deviceID, &deviceHostname); err != nil {
				http.Error(w, fmt.Sprintf("Error scanning device ID and hostname: %v", err), http.StatusInternalServerError)
				return
			}
			deviceMap[deviceID] = deviceHostname
		}
	} else {
		// Translate device names to device IDs
		queryPlaceholders := strings.Repeat("?,", len(devices))
		queryPlaceholders = strings.TrimSuffix(queryPlaceholders, ",")
		deviceNameQuery := fmt.Sprintf("SELECT device_id, device_hostname FROM `%s`.Devices WHERE TRIM(device_hostname) IN (%s)", dbName, queryPlaceholders)

		args := make([]interface{}, len(devices))
		for i, device := range devices {
			args[i] = device
		}

		rows, err := db.Query(deviceNameQuery, args...)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error querying device names: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var deviceID, deviceHostname string
			if err := rows.Scan(&deviceID, &deviceHostname); err != nil {
				http.Error(w, fmt.Sprintf("Error scanning device ID and hostname: %v", err), http.StatusInternalServerError)
				return
			}
			deviceMap[deviceID] = deviceHostname
		}
	}

	// Fill deviceIDs array

	deviceIDs := make([]string, 0, len(deviceMap))

	for deviceID := range deviceMap {
		deviceIDs = append(deviceIDs, deviceID)
	}

	// Construct the query for performance metrics
	if len(deviceIDs) == 0 {
		http.Error(w, "No device IDs found for the given device names", http.StatusBadRequest)
		return
	}

	// Get the top process IDs using the new function
	devicePIDsMap, err := helpers.GetTopProcessIDs(db, dbName, deviceIDs, timeStart, timeEnd, numberOfProcesses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If no processes are identified, return an empty response
	if len(devicePIDsMap) == 0 {
		fmt.Println("No processes were found when querying CPU metrics")
		w.Header().Set("Content-Type", "application/json")

		response := map[string]string{
			"message": "No processes were found when querying CPU metrics",
		}

		responseJSON, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Error creating JSON response", http.StatusInternalServerError)
			return
		}

		w.Write(responseJSON)
		return
	}

	// DS to hold metrics per device
	deviceMetricsMap := make(map[string][]CPUMetricsResponse)
	// Construct the query for performance metrics for the identified top processes
	for deviceID, pids := range devicePIDsMap {

		performanceQuery := fmt.Sprintf(`
		SELECT 
			pm.timestamp,
			psm.process_pid,
			psm.process_name,
			psm.process_command,
			process_cpu_usage
		FROM 
			%s.PerformanceMetrics pm
		JOIN 
			%s.ProcessMetrics psm ON pm.metric_id = psm.metric_id
		WHERE 
			pm.device_id = ?
			AND psm.process_pid IN (%s)
			AND pm.timestamp BETWEEN ? AND ?
		ORDER BY 
			pm.timestamp
	`, fmt.Sprintf("`%s`", dbName), fmt.Sprintf("`%s`", dbName), strings.Trim(strings.Repeat("?,", len(pids)), ","))

		// Prepare the arguments for the performanceQuery
		args := []interface{}{deviceID}
		for _, pid := range pids {
			args = append(args, pid)
		}
		args = append(args, timeStart, timeEnd)

		rows, err := db.Query(performanceQuery, args...)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error querying CPU performance metrics: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Parse and MAP the query results
		for rows.Next() {
			var metric CPUMetricsResponse
			if err := rows.Scan(&metric.Timestamp, &metric.ProcessPID, &metric.ProcessName, &metric.ProcessCommand, &metric.ProcessCPUUsage); err != nil {
				http.Error(w, fmt.Sprintf("Error scanning row: %v", err), http.StatusInternalServerError)
				return
			}
			deviceMetricsMap[deviceID] = append(deviceMetricsMap[deviceID], metric)
		}

	}

	// BEGIN NEW CODE
	// Group Metrics by process name for each device ID
	var deviceMetrics []CpuDeviceMetrics
	for deviceID, metrics := range deviceMetricsMap {
		processGroups := make(map[string][]CPUMetricsResponse)
		cpuUsageTotals := make(map[string]float64)
		cpuUsageCounts := make(map[string]int)

		// Group metrics by process name
		for _, metric := range metrics {
			processName := metric.ProcessName
			processGroups[processName] = append(processGroups[processName], metric)
			cpuUsageTotals[processName] += metric.ProcessCPUUsage
			cpuUsageCounts[processName]++
		}

		// Calculate the average CPU usage for each process
		groupedMetrics := make([]CpuProcessGroup, 0, len(processGroups))
		for name, metrics := range processGroups {
			avgCpu := int64(cpuUsageTotals[name] / float64(cpuUsageCounts[name]))
			groupedMetrics = append(groupedMetrics, CpuProcessGroup{
				ProcessName: name,
				AvgCpu:      avgCpu,
				Metrics:     metrics,
			})
		}

		// Sort the grouped metrics by average CPU usage
		sort.Slice(groupedMetrics, func(i, j int) bool {
			return groupedMetrics[i].AvgCpu > groupedMetrics[j].AvgCpu
		})

		// Append sorted metrics to response
		deviceMetrics = append(deviceMetrics, CpuDeviceMetrics{
			DeviceID:   deviceID,
			DeviceName: deviceMap[deviceID],
			Metrics:    groupedMetrics,
		})
	}

	// END NEW CODE

	// Encode the structured response as JSON and send it to the client.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deviceMetrics); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		return
	}
}
