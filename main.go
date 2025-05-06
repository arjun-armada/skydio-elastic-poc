package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gocql/gocql"
	"github.com/olivere/elastic/v7"
)

type FlightMedia struct {
	DownloadURL string `json:"download_url"`
	FlightID    string `json:"flight_id"`
	//MediaUUID   string `json:"media_uuid"`
	CapturedTime int64 `json:"captured_time"`
}

// type Flight struct {
// 	FlightID   string  `json:"flight_id"`
// 	TakeoffLat float64 `json:"takeoff_lat"`
// 	TakeoffLon float64 `json:"takeoff_long"`
// }

type FlightTelemetry struct {
	FlightID  string  `json:"flight_id"`
	GPSLat    float64 `json:"gps_latitude"`
	GPSLon    float64 `json:"gps_longitude"`
	Timestamp int64   `json:"timestamp"`
}

type JoinedData struct {
	FlightMedia FlightMedia `json:"FlightMedia"`
	//Flights     Flight      `json:"Flights"`
	Location struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"location"`
}

func main() {
	cluster := gocql.NewCluster("127.0.0.1:12127") // change to your Scylla IP or DNS
	cluster.Keyspace = "skydiodb"
	cluster.Consistency = gocql.Quorum

	session, err := cluster.CreateSession()
	if err != nil {
		log.Fatalf("Failed to connect to ScyllaDB: %v", err)
	}
	defer session.Close()

	fmt.Println("Connected to scyllaDB!!")
	var listFlightMedia = make(map[string][]FlightMedia)
	batch := gocql.Batch{}

	iter := session.Query(`SELECT flight_id, download_url, tounixtimestamp(captured_time) FROM flight_media`).Iter()
	var fm FlightMedia
	var mediaCount int
	for iter.Scan(&fm.FlightID, &fm.DownloadURL, &fm.CapturedTime) {
		listFlightMedia[fm.FlightID] = append(listFlightMedia[fm.FlightID], fm)
		batch.Entries = append(batch.Entries, gocql.BatchEntry{
			Stmt:       fmt.Sprintf("SELECT flight_id, gps_latitude, gps_longitude, tounixtimestamp(timestamp) FROM flight_telemetry where flight_id='%s' and  timestamp <= %d order by timestamp desc limit 1;", fm.FlightID, fm.CapturedTime),
			Idempotent: true,
		})
		mediaCount++
	}

	if err := iter.Close(); err != nil {
		log.Fatalf("Error fetching flightMedia: %v", err)
	}

	fmt.Println("mediaCount: ", mediaCount)

	err = session.ExecuteBatch(&batch)
	if err != nil {
		log.Fatal("error executing the batch query", err)
	}

	var joinedData []JoinedData

	// iter = session.Query(`SELECT flight_id, gps_latitude, gps_longitude, tounixtimestamp(timestamp) FROM flight_telemetry where flight_id='4389b50b-866f-46e3-bfc1-62a75a80b784' and  timestamp <= 1712082346764 order by timestamp desc limit 1;`).Iter()

	// iter = session.Query(`SELECT flight_id, gps_latitude, gps_longitude, tounixtimestamp(timestamp) FROM flight_telemetry`).Iter()
	var f FlightTelemetry
	for iter.Scan(&f.FlightID, &f.GPSLat, &f.GPSLon, &f.Timestamp) {
		if medias, exists := listFlightMedia[f.FlightID]; exists {

			for _, v := range medias {
				joinedData = append(joinedData, JoinedData{FlightMedia: v, Location: struct {
					Lat float64 `json:"lat"`
					Lon float64 `json:"lon"`
				}{
					Lat: f.TakeoffLat,
					Lon: f.TakeoffLon,
				}})
			}

		}
	}

	if err := iter.Close(); err != nil {
		log.Fatalf("Error fetching orders: %v", err)
	}

	fmt.Println("len(joinedData): ", len(joinedData))
	pushToElasticsearch(joinedData)
}

func pushToElasticsearch(data []JoinedData) {
	// Connect to Elasticsearch
	esURL := "http://localhost:9200"
	client, err := elastic.NewClient(elastic.SetURL(esURL), elastic.SetSniff(false))
	if err != nil {
		log.Fatalf("Error creating Elasticsearch client: %v", err)
	}

	indexName := "flight-media-join-flights"

	// Create index if not exists
	exists, err := client.IndexExists(indexName).Do(context.Background())
	if err != nil {
		log.Fatalf("Error checking index existence: %v", err)
	}
	if !exists {
		_, err := client.CreateIndex(indexName).BodyString(`{
			"mappings": {
						"properties": {
							"location": { "type": "geo_point" }
						}
			}
		}`).Do(context.Background())
		if err != nil {
			log.Fatalf("Error creating index: %v", err)
		}
		fmt.Println("Index created successfully!")
	}

	// Insert data
	for _, item := range data {
		_, err := client.Index().
			Index(indexName).
			BodyJson(item).
			Do(context.Background())
		if err != nil {
			log.Printf("Failed to insert document: %v", err)
			break
		}
	}

	fmt.Println("Data successfully pushed to Elasticsearch!")
}
