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
}

type Flight struct {
	FlightID   string  `json:"flight_id"`
	TakeoffLat float64 `json:"takeoff_lat"`
	TakeoffLon float64 `json:"takeoff_long"`
}

type JoinedData struct {
	FlightMedia FlightMedia `json:"FlightMedia"`
	Flights     Flight      `json:"Flights"`
	Location    struct {
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

	iter := session.Query(`SELECT flight_id,  download_url FROM flight_media`).Iter()
	var fm FlightMedia
	var mediaCount int
	for iter.Scan(&fm.FlightID, &fm.DownloadURL) {
		listFlightMedia[fm.FlightID] = append(listFlightMedia[fm.FlightID], fm)
		mediaCount++
	}
	if err := iter.Close(); err != nil {
		log.Fatalf("Error fetching flightMedia: %v", err)
	}

	fmt.Println("mediaCount: ", mediaCount)

	var joinedData []JoinedData
	iter = session.Query(`SELECT flight_id, takeoff_lat, takeoff_long FROM flights`).Iter()
	var f Flight
	for iter.Scan(&f.FlightID, &f.TakeoffLat, &f.TakeoffLon) {
		if medias, exists := listFlightMedia[f.FlightID]; exists {

			for _, v := range medias {
				joinedData = append(joinedData, JoinedData{FlightMedia: v, Flights: f, Location: struct {
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
