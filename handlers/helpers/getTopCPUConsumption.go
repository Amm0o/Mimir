package helpers

import (
	"database/sql"
	"fmt"
	"strings"
)

func GetTopProcessIDs(db *sql.DB, dbName string, deviceIDs []string, timeStart string, timeEnd string, numberOfProcesses int) ([]int, error) {
	// Construct the query to get the top N processes based on total CPU usage
	queryPlaceholders := strings.Repeat("?,", len(deviceIDs))
	queryPlaceholders = strings.TrimSuffix(queryPlaceholders, ",")
	topProcessesQuery := fmt.Sprintf(`
	    SELECT 
	        psm.process_pid
	    FROM 
	        %s.PerformanceMetrics pm
	    JOIN 
	        %s.ProcessMetrics psm ON pm.metric_id = psm.metric_id
	    WHERE 
	        pm.device_id IN (%s)
	        AND pm.timestamp BETWEEN ? AND ?
	    GROUP BY 
	        psm.process_pid
	    ORDER BY 
	        SUM(psm.process_cpu_usage) DESC
	`, fmt.Sprintf("`%s`", dbName), fmt.Sprintf("`%s`", dbName), queryPlaceholders)

	// Add LIMIT clause only if numberOfProcesses is greater than 0
	if numberOfProcesses > 0 {
		topProcessesQuery += fmt.Sprintf(" LIMIT %d", numberOfProcesses)
	}

	// Prepare the arguments for the topProcessesQuery
	args := make([]interface{}, len(deviceIDs)+2)
	for i, id := range deviceIDs {
		args[i] = id
	}
	args[len(deviceIDs)] = timeStart
	args[len(deviceIDs)+1] = timeEnd

	// Execute the subquery to get the top N processes
	rows, err := db.Query(topProcessesQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying top processes: %v", err)
	}
	defer rows.Close()

	// Collect the top process IDs
	var topProcessIDs []int
	for rows.Next() {
		var processPID int
		if err := rows.Scan(&processPID); err != nil {
			return nil, fmt.Errorf("error scanning process PID: %v", err)
		}
		topProcessIDs = append(topProcessIDs, processPID)
	}

	return topProcessIDs, nil
}
