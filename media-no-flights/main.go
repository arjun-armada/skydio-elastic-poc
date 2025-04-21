package main

import (
	"fmt"
	"log"

	"github.com/gocql/gocql"
)

func main() {
	cluster := gocql.NewCluster("127.0.0.1:12127") // change to your Scylla IP or DNS
	cluster.Keyspace = "skydiodb"
	cluster.Consistency = gocql.Quorum

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("Failed to connect to ScyllaDB: %v", err)
	}
	defer session.Close()

	// Step 1: Load all flight_ids from flight table into a map
	validFlightIDs := make(map[string]struct{})

	iter := session.Query("SELECT flight_id FROM flights").Iter()
	var fid string
	for iter.Scan(&fid) {
		validFlightIDs[fid] = struct{}{}
	}
	if err := iter.Close(); err != nil {
		log.Fatalf("Error reading flight table: %v", err)
	}
	fmt.Printf("Loaded %d valid flight_ids\n", len(validFlightIDs))

	// Scan flight_media table and count orphaned rows
	var orphanCount int
	var validCount int
	iter = session.Query("SELECT flight_id FROM flight_media").Iter()
	for iter.Scan(&fid) {
		if _, exists := validFlightIDs[fid]; !exists {
			orphanCount++
		} else {
			validCount++
		}
	}
	if err := iter.Close(); err != nil {
		log.Fatalf("Error reading flight_media table: %v", err)
	}

	fmt.Printf("Found %d orphaned flight_media rows (no matching flight_id)\n", orphanCount)
	fmt.Printf("Found %d valid flight_media rows \n", validCount)
}
