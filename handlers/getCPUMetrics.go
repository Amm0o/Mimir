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
	ProcessRAMUsage int64   `json:"processRamUsage"`
	AvgCpuUsage     float64 `json:"avgCpuUsage"`
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
	var deviceIDs []string
	if len(devices) == 0 {
		deviceQuery := fmt.Sprintf("SELECT device_id FROM `%s`.Devices", dbName)
		rows, err := db.Query(deviceQuery)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error querying devices: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var deviceID string
			if err := rows.Scan(&deviceID); err != nil {
				http.Error(w, fmt.Sprintf("Error scanning device ID: %v", err), http.StatusInternalServerError)
				return
			}
			deviceIDs = append(deviceIDs, deviceID)
		}
	} else {
		// Translate device names to device IDs
		queryPlaceholders := strings.Repeat("?,", len(devices))
		queryPlaceholders = strings.TrimSuffix(queryPlaceholders, ",")
		deviceNameQuery := fmt.Sprintf("SELECT device_id FROM `%s`.Devices WHERE TRIM(device_hostname) IN (%s)", dbName, queryPlaceholders)

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
			var deviceID string
			if err := rows.Scan(&deviceID); err != nil {
				http.Error(w, fmt.Sprintf("Error scanning device ID: %v", err), http.StatusInternalServerError)
				return
			}
			deviceIDs = append(deviceIDs, deviceID)
		}
	}

	// Construct the query for performance metrics
	if len(deviceIDs) == 0 {
		http.Error(w, "No device IDs found for the given device names", http.StatusBadRequest)
		return
	}

	// Get the top process IDs using the new function
	topProcessIDs, err := helpers.GetTopProcessIDs(db, dbName, deviceIDs, timeStart, timeEnd, numberOfProcesses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If no processes are identified, return an empty response
	if len(topProcessIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	// Construct the query for performance metrics for the identified top processes
	queryPlaceholders := strings.Repeat("?,", len(deviceIDs))
	queryPlaceholders = strings.TrimSuffix(queryPlaceholders, ",")
	processIDPlaceholders := strings.Repeat("?,", len(topProcessIDs))
	processIDPlaceholders = strings.TrimSuffix(processIDPlaceholders, ",")
	performanceQuery := fmt.Sprintf(`
	SELECT 
		pm.timestamp,
		psm.process_pid,
		psm.process_name,
		psm.process_command,
		psm.process_cpu_usage,
		psm.process_ram_usage
	FROM 
		%s.PerformanceMetrics pm
	JOIN 
		%s.ProcessMetrics psm ON pm.metric_id = psm.metric_id
	WHERE 
		pm.device_id IN (%s)
		AND psm.process_pid IN (%s)
		AND pm.timestamp BETWEEN ? AND ?
	ORDER BY 
		pm.timestamp
`, fmt.Sprintf("`%s`", dbName), fmt.Sprintf("`%s`", dbName), queryPlaceholders, processIDPlaceholders)

	// Prepare the arguments for the performanceQuery
	args := make([]interface{}, len(deviceIDs)+len(topProcessIDs)+2)
	for i, id := range deviceIDs {
		args[i] = id
	}
	for i, pid := range topProcessIDs {
		args[len(deviceIDs)+i] = pid
	}
	args[len(deviceIDs)+len(topProcessIDs)] = timeStart
	args[len(deviceIDs)+len(topProcessIDs)+1] = timeEnd

	rows, err := db.Query(performanceQuery, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error querying performance metrics: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Parse the query results
	var cpuMetrics []CPUMetricsResponse
	for rows.Next() {
		var metric CPUMetricsResponse
		if err := rows.Scan(&metric.Timestamp, &metric.ProcessPID, &metric.ProcessName, &metric.ProcessCommand, &metric.ProcessCPUUsage, &metric.ProcessRAMUsage); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning row: %v", err), http.StatusInternalServerError)
			return
		}
		cpuMetrics = append(cpuMetrics, metric)
	}

	// Aggregate metrics by ProcessName, then by ProcessPID.
	processMetricsMap := make(map[string]*ProcessMetrics)
	for _, metric := range cpuMetrics {
		processName := metric.ProcessName
		pid := metric.ProcessPID

		if _, exists := processMetricsMap[processName]; !exists {
			processMetricsMap[processName] = &ProcessMetrics{
				ProcessName:         processName,
				TotalCPUConsumption: 0,
				Metrics:             make(map[int][]CPUMetricsResponse),
			}
		}

		processMetric := processMetricsMap[processName]
		processMetric.TotalCPUConsumption += metric.ProcessCPUUsage
		processMetric.Metrics[pid] = append(processMetric.Metrics[pid], metric)
	}

	// Convert map to slice for sorting.
	var processMetricsSlice []ProcessMetrics
	for _, v := range processMetricsMap {
		processMetricsSlice = append(processMetricsSlice, *v)
	}

	// Sort the slice by TotalCPUConsumption in descending order.
	sort.Slice(processMetricsSlice, func(i, j int) bool {
		return processMetricsSlice[i].TotalCPUConsumption > processMetricsSlice[j].TotalCPUConsumption
	})

	// Prepare the final structured response.
	var finalResponse []ProcessGroup
	for _, processMetric := range processMetricsSlice {
		for pid, metrics := range processMetric.Metrics {
			avgCPU := processMetric.TotalCPUConsumption / float64(len(metrics))
			finalResponse = append(finalResponse, ProcessGroup{
				ProcessName: processMetric.ProcessName,
				ProcessPID:  pid,
				AvgCPU:      avgCPU,
				Metrics:     metrics,
			})
		}
	}

	// Sort the final response by AvgCPU in descending order.
	sort.Slice(finalResponse, func(i, j int) bool {
		return finalResponse[i].AvgCPU > finalResponse[j].AvgCPU
	})

	// Encode the structured response as JSON and send it to the client.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(finalResponse); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		return
	}
}
