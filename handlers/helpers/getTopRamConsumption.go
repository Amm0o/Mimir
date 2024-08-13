package helpers

import (
	"database/sql"
	"fmt"
)

func GetTopRamProcessIDs(db *sql.DB, dbName string, deviceIDs []string, timeStart string, timeEnd string, numberOfProcesses int) (map[string][]int, error) {
	// Define a map to hold the device IDs and their corresponding arrays of PIDs
	devicePIDsMap := make(map[string][]int)

	// Iterate over each device ID to query the PIDs
	for _, deviceID := range deviceIDs {
		// Construct the query to get the top N processes based on total RAM usage for the current device ID
		topProcessesQuery := fmt.Sprintf(`
        SELECT 
            psm.process_pid
        FROM 
            %s.PerformanceMetrics pm
        JOIN 
            %s.ProcessMetrics psm ON pm.metric_id = psm.metric_id
        WHERE 
            pm.device_id = ?
            AND pm.timestamp BETWEEN ? AND ?
        GROUP BY 
            psm.process_pid
        ORDER BY 
            SUM(psm.process_ram_usage) DESC
    `, fmt.Sprintf("`%s`", dbName), fmt.Sprintf("`%s`", dbName))

		// Add LIMIT clause only if numberOfProcesses is greater than 0
		if numberOfProcesses > 0 {
			topProcessesQuery += fmt.Sprintf(" LIMIT %d", numberOfProcesses)
		}

		// Prepare the arguments for the topProcessesQuery
		args := []interface{}{deviceID, timeStart, timeEnd}

		// Execute the query to get the top N processes for the current device ID
		rows, err := db.Query(topProcessesQuery, args...)
		if err != nil {
			return nil, fmt.Errorf("error querying top processes for device %s: %v", deviceID, err)
		}
		defer rows.Close()

		// Collect the top process IDs for the current device ID
		var pids []int
		for rows.Next() {
			var processPID int
			if err := rows.Scan(&processPID); err != nil {
				return nil, fmt.Errorf("error scanning process PID for device %s: %v", deviceID, err)
			}
			pids = append(pids, processPID)
		}

		// Store the collected PIDs in the map
		devicePIDsMap[deviceID] = pids
	}

	return devicePIDsMap, nil
}
