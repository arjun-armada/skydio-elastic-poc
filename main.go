package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gocql/gocql"
	"github.com/olivere/elastic/v7"
	"github.com/scylladb/gocqlx/v3"
)

type FlightMedia struct {
	DownloadURL string `json:"download_url"`
	FlightID    string `json:"flight_id"`
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
	// Connect to ScyllaDB
	// cluster := gocql.NewCluster("10.2.223.174")
	// cluster.Keyspace = "skydiodb"
	// cluster.Consistency = gocql.Quorum
	// session, err := cluster.CreateSession()
	// if err != nil {
	// 	log.Fatalf("Error connecting to ScyllaDB: %v", err)
	// }

	// fmt.Println(session)

	// defer session.Close()

	clusterConfig := *gocql.NewCluster([]string{"127.0.0.1:12127", "127.0.0.1:12128", "127.0.0.1:12129"}...)
	// clusterConfig.Hosts = []string{"127.0.0.1:12127", "127.0.0.1:12128", "127.0.0.1:12129"}
	clusterConfig.Timeout = 60 * time.Second
	clusterConfig.ConnectTimeout = 5 * time.Second
	clusterConfig.Keyspace = "skydiodb"

	// clusterConfig.SerialConsistency = gocql.LocalSerial
	// clusterConfig.Consistency = gocql.Quorum // You can adjust based on your use case

	// // Round-robin load balancing policy (spread queries across all nodes)
	// clusterConfig.PoolConfig.HostSelectionPolicy = gocql.RoundRobinHostPolicy()

	var err error
	_, err = gocqlx.WrapSession(gocql.NewSession(clusterConfig))

	if err != nil {
		log.Fatalln("Error opening session to skydiodb", err)
	}
	fmt.Println("Connected to scyllaDB!!")
	session, _ := clusterConfig.CreateSession()
	//fmt.Println(session)

	// // Fetch flightMedia
	var listFlightMedia = make(map[string]FlightMedia)
	iter := session.Query(`SELECT flight_id, download_url FROM flight_media`).Iter()
	var fm FlightMedia
	for iter.Scan(&fm.FlightID, &fm.DownloadURL) {
		listFlightMedia[fm.FlightID] = fm
	}
	if err := iter.Close(); err != nil {
		log.Fatalf("Error fetching flightMedia: %v", err)
	}

	fmt.Println(len(listFlightMedia))

	// var listFlights = make(map[string]Flight)
	// iter = session.Query(`SELECT flight_id, takeoff_lat, takeoff_long FROM flights`).Iter()
	// var f Flight
	// for iter.Scan(&f.FlightID, &f.TakeoffLat, &f.TakeoffLon) {
	// 	listFlights[f.FlightID] = f
	// }

	// fmt.Println("len of listFlights: ", len(listFlights))

	// Fetch orders and join with users
	var joinedData []JoinedData
	iter = session.Query(`SELECT flight_id, takeoff_lat, takeoff_long FROM flights`).Iter()
	var f Flight
	for iter.Scan(&f.FlightID, &f.TakeoffLat, &f.TakeoffLon) {
		if fm, exists := listFlightMedia[f.FlightID]; exists {
			joinedData = append(joinedData, JoinedData{FlightMedia: fm, Flights: f, Location: struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			}{
				Lat: f.TakeoffLat,
				Lon: f.TakeoffLon,
			}})
		}
	}

	fmt.Println(len(joinedData))

	fmt.Println("joined[0]: ", joinedData[0].Flights.TakeoffLat, joinedData[0].Flights.TakeoffLon)

	if err := iter.Close(); err != nil {
		log.Fatalf("Error fetching orders: %v", err)
	}

	// // Push joined data to Elasticsearch
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
		}
	}

	fmt.Println("Data successfully pushed to Elasticsearch!")
}
