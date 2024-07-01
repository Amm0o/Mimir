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

type RamMetricsRequest struct {
	TenantID string `json:"tenantID"`
	Query    Query  `json:"query"`
}

type RamMetricsReponse struct {
	Timestamp       string `json:"timestamp"`
	ProcessPID      int    `json:"processPID"`
	ProcessName     string `json:"processName"`
	ProcessCommand  string `json:"processCommand"`
	ProcessRamUsage int64  `json:"processRamUsage"`
}

type RamProcessGroup struct {
	ProcessName string              `json:"processName"`
	AvgRam      int64               `json:"avgRam"`
	Metrics     []RamMetricsReponse `json:"metrics"`
}

// Function to handle the retrieval of RAM metrics
func RetrieveRamMetrics(w http.ResponseWriter, r *http.Request) {
	var ramMetricsRequest RamMetricsRequest

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Parse the JSON data
	if err := json.Unmarshal(body, &ramMetricsRequest); err != nil {
		http.Error(w, "Invalid JSON data", http.StatusBadRequest)
		fmt.Println(err)
		return
	}

	tenantID := ramMetricsRequest.TenantID
	devices := ramMetricsRequest.Query.Devices
	timeStart := ramMetricsRequest.Query.TimeRange.Start
	timeEnd := ramMetricsRequest.Query.TimeRange.End
	numberOfProcesses := ramMetricsRequest.Query.NumberOfProcesses

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

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get the top process IDs using the new function
	topProcessIDs, err := helpers.GetTopRamProcessIDs(db, dbName, deviceIDs, timeStart, timeEnd, numberOfProcesses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If no processes are identified, return an empty response
	if len(deviceIDs) == 0 {
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
		http.Error(w, fmt.Sprintf("Error querying RAM performance metrics: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Parse the query results
	var ramMetrics []RamMetricsReponse
	for rows.Next() {
		var metric RamMetricsReponse
		if err := rows.Scan(&metric.Timestamp, &metric.ProcessPID, &metric.ProcessName, &metric.ProcessCommand, &metric.ProcessRamUsage); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning row: %v", err), http.StatusInternalServerError)
			return
		}
		ramMetrics = append(ramMetrics, metric)
	}

	// Aggregate metrics by process name.
	processGroups := make(map[string][]RamMetricsReponse)
	ramUsageTotals := make(map[string]int64)
	ramUsageCounts := make(map[string]int)

	for _, metric := range ramMetrics {
		processName := metric.ProcessName // Minimize map lookups by storing repeated values in variables
		processGroups[processName] = append(processGroups[processName], metric)
		ramUsageTotals[processName] += metric.ProcessRamUsage
		ramUsageCounts[processName]++
	}

	// Preallocate groupedMetrics with the size of processGroups to avoid reallocation.
	groupedMetrics := make([]RamProcessGroup, 0, len(processGroups))

	for name, metrics := range processGroups {
		avgRam := ramUsageTotals[name] / int64(ramUsageCounts[name])
		groupedMetrics = append(groupedMetrics, RamProcessGroup{
			ProcessName: name,
			AvgRam:      avgRam,
			Metrics:     metrics,
		})
	}

	// Sort the grouped metrics by avgRam in descending order.
	sort.Slice(groupedMetrics, func(i, j int) bool {
		return groupedMetrics[i].AvgRam > groupedMetrics[j].AvgRam
	})

	// Encode the structured response as JSON and send it to the client.
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(groupedMetrics); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		return
	}
}
