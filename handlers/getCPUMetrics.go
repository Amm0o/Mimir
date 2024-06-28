package handlers

import (
	"cloudVigilante/backend/handlers/helpers"
	"encoding/json"
	"fmt"
	"io"
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
	AvgCPU      float64              `json:"avgCpu"`
	Metrics     []CPUMetricsResponse `json:"metrics"`
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

	// Determine the database name
	dbName := fmt.Sprintf("Performance_%s", tenantID)

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

	// Aggregate metrics by process name.
	processGroups := make(map[string][]CPUMetricsResponse)
	cpuUsageTotals := make(map[string]float64)
	cpuUsageCounts := make(map[string]int)

	for _, metric := range cpuMetrics {
		processName := metric.ProcessName // Minimize map lookups by storing repeated values in variables
		processGroups[processName] = append(processGroups[processName], metric)
		cpuUsageTotals[processName] += metric.ProcessCPUUsage
		cpuUsageCounts[processName]++
	}

	// Preallocate groupedMetrics with the size of processGroups to avoid reallocation.
	groupedMetrics := make([]ProcessGroup, 0, len(processGroups))

	for name, metrics := range processGroups {
		avgCPU := cpuUsageTotals[name] / float64(cpuUsageCounts[name])
		groupedMetrics = append(groupedMetrics, ProcessGroup{
			ProcessName: name,
			AvgCPU:      avgCPU,
			Metrics:     metrics,
		})
	}

	// Sort the grouped metrics by AvgCPU in descending order.
	sort.Slice(groupedMetrics, func(i, j int) bool {
		return groupedMetrics[i].AvgCPU > groupedMetrics[j].AvgCPU
	})

	// Encode the structured response as JSON and send it to the client.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(groupedMetrics); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		return
	}
}
