package models

import (
	"database/sql"
	"fmt"
	"log"
)

// DB data structures

type DeviceData struct {
	DeviceID   string
	Hostname   string
	MACAddress string
	IPAddress  string
}

type PerformanceData struct {
	DeviceID    string
	Timestamp   string
	CPUUsage    float64
	RAMUsage    int64
	DiskUsage   int64
	TotalMemory int64
	UsedMemoryP float64
	Processes   []ProcessData
}

type ProcessData struct {
	PID      int
	Name     string
	Command  string
	CPUUsage float64
	RAMUsage int64
}

// Handle creating the performance db
func CreatePerformanceDB(db *sql.DB, orgID string) error {

	// Format db name and create it
	dbName := fmt.Sprintf("Peformance_%s", orgID)

	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))

	if err != nil {
		return err
	} else {
		log.Printf("Created table %s", dbName)
	}

	// Use the db just created
	_, err = db.Exec(fmt.Sprintf("USE %s", dbName))
	if err != nil {
		return err
	}

	// logic to create the tables
	createTablesQuery := []string{
		`CREATE TABLE IF NOT EXISTS Devices (
            id INT AUTO_INCREMENT PRIMARY KEY,
            device_id VARCHAR(255) UNIQUE NOT NULL,
            device_hostname VARCHAR(255),
            mac_address VARCHAR(255),
            ip_address VARCHAR(255)
        )`,
		`CREATE TABLE IF NOT EXISTS PerformanceMetrics (
            metric_id INT AUTO_INCREMENT PRIMARY KEY,
            device_id VARCHAR(255) NOT NULL,
            timestamp DATETIME NOT NULL,
            cpu_usage FLOAT,
            ram_usage BIGINT,
            disk_usage BIGINT,
            FOREIGN KEY (device_id) REFERENCES Devices(device_id)
        )`,
		`CREATE TABLE IF NOT EXISTS ProcessMetrics (
            process_metric_id INT AUTO_INCREMENT PRIMARY KEY,
            metric_id INT NOT NULL,
            process_pid INT,
            process_name VARCHAR(255),
            process_command TEXT,
            process_cpu_usage FLOAT,
            process_ram_usage BIGINT,
            FOREIGN KEY (metric_id) REFERENCES PerformanceMetrics(metric_id)
        )`,
	}

	for _, query := range createTablesQuery {

		_, err := db.Exec(query)
		if err != nil {
			return nil
		}

		log.Println("Create db and tables successfully")

	}

	// In case all went well:
	return nil
}

// Handle new performance data coming in
func InsertPerformanceData(db *sql.DB, orgID string, deviceData DeviceData, perfData PerformanceData) error {

	// Select the correct db
	dbName := fmt.Sprintf("PerformanceDB_%s", orgID)
	_, err := db.Exec(fmt.Sprintf("USE %s", dbName))

	// Insert device data if it does not exist
	insertDeviceQuery := `INSERT INTO Devices (device_id, device_hostname, mac_address, ip_address) 
                          VALUES (?, ?, ?, ?)
                          ON DUPLICATE KEY UPDATE device_hostname=VALUES(device_hostname), mac_address=VALUES(mac_address), ip_address=VALUES(ip_address)`

	_, err = db.Exec(insertDeviceQuery, deviceData.DeviceID, deviceData.Hostname, deviceData.MACAddress, deviceData.IPAddress)
	if err != nil {
		return err
	}

	log.Println("Inserted device data successfully")

	// Insert performance metrics
	insertPerfQuery := `INSERT INTO PerformanceMetrics (device_id, timestamp, cpu_usage, ram_usage, disk_usage) 
                        VALUES (?, ?, ?, ?, ?)`

	result, err := db.Exec(insertPerfQuery, perfData.DeviceID, perfData.Timestamp, perfData.CPUUsage, perfData.RAMUsage, perfData.DiskUsage)
	if err != nil {
		return err
	}

	log.Println("Inserted performance metrics successfully")

	// Get Metric ID from overall performance metric
	metricID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	log.Println("Got metricID")

	// Insert process metrics
	for _, process := range perfData.Processes {
		insertProcessQuery := `INSERT INTO ProcessMetrics (metric_id, process_pid, process_name, process_command, process_cpu_usage, process_ram_usage) 
		VALUES (?, ?, ?, ?, ?, ?)`

		_, err := db.Exec(insertProcessQuery, metricID, process.PID, process.Name, process.Command, process.CPUUsage, process.RAMUsage)
		if err != nil {
			return err
		}

		log.Println("Inserted process performance metrics successfully")
	}

	// all went ok
	return nil
}
