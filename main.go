package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/gocql/gocql"
	"github.com/olivere/elastic/v7"
)

type FlightMedia struct {
	DownloadURL string `json:"download_url"`
	FlightID    string `json:"flight_id"`
	//MediaUUID   string `json:"media_uuid"`
	CapturedTime int64 `json:"captured_time"`
}

type Flight struct {
	FlightID   string  `json:"flight_id"`
	TakeoffLat float64 `json:"takeoff_lat"`
	TakeoffLon float64 `json:"takeoff_long"`
}

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

	iter := session.Query(`SELECT flight_id, gps_latitude, gps_longitude, tounixtimestamp(timestamp) FROM flight_telemetry`).Iter()
	var mapFlightTelemetry = make(map[string][]FlightTelemetry)
	var ft FlightTelemetry
	for iter.Scan(&ft.FlightID, &ft.GPSLat, &ft.GPSLon, &ft.Timestamp) {
		mapFlightTelemetry[ft.FlightID] = append(mapFlightTelemetry[ft.FlightID], ft)
	}

	// Sorting the Slices for Binary search.
	for flightID, telemetryList := range mapFlightTelemetry {
		sort.Slice(telemetryList, func(i, j int) bool {
			return telemetryList[i].Timestamp < telemetryList[j].Timestamp
		})
		mapFlightTelemetry[flightID] = telemetryList // reassign the sorted slice
	}

	iter = session.Query(`SELECT flight_id, takeoff_lat, takeoff_long FROM flights`).Iter()
	var mapFlights = make(map[string]Flight)
	var flight Flight
	for iter.Scan(&flight.FlightID, &flight.TakeoffLat, &flight.TakeoffLon) {
		mapFlights[flight.FlightID] = flight
	}

	var joinedData []JoinedData

	iter = session.Query(`SELECT flight_id, download_url, tounixtimestamp(captured_time) FROM flight_media`).Iter()
	var fm FlightMedia
	for iter.Scan(&fm.FlightID, &fm.DownloadURL, &fm.CapturedTime) {
		if ft, exists := mapFlightTelemetry[fm.FlightID]; exists {
			index := getLatLonFlightTelemetry(ft, fm.CapturedTime)
			joinedData = append(joinedData, JoinedData{FlightMedia: fm, Location: struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			}{
				Lat: ft[index].GPSLat,
				Lon: ft[index].GPSLon,
			}})
		} else if flight, exists := mapFlights[fm.FlightID]; exists {
			joinedData = append(joinedData, JoinedData{FlightMedia: fm, Location: struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			}{
				Lat: flight.TakeoffLat,
				Lon: flight.TakeoffLat,
			}})
		} else {
			joinedData = append(joinedData, JoinedData{FlightMedia: fm, Location: struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			}{
				Lat: 0,
				Lon: 0,
			}})
		}
	}

	fmt.Println("len(joinedData): ", len(joinedData))

	pushToElasticsearchWithAPIKey(joinedData)
	//pushToElasticsearch(joinedData)
}

func getLatLonFlightTelemetry(ft []FlightTelemetry, capturedTime int64) int {
	left := 0
	right := len(ft) - 1
	ans := -1
	for left <= right {
		mid := left + (right-left)/2
		if ft[mid].Timestamp <= capturedTime {
			ans = mid
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return ans
}

type transportWithAPIKey struct {
	apiKeyHeader string
	rt           http.RoundTripper
}

func (t *transportWithAPIKey) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.apiKeyHeader)
	return t.rt.RoundTrip(req)
}

func pushToElasticsearchWithAPIKey(data []JoinedData) {
	esURL := "https://quickstart-es-http.default.svc:9200"
	apiKeyHeader := "ApiKey " + "amtxSnBwWUJpa2VaMVprU2pEN0g6MjFSb1JPODNTMkdvVGlocVFSVmxnUQ=="

	// Load your CA cert
	caCert, err := os.ReadFile("eck-ca.crt")
	if err != nil {
		log.Fatalf("Failed to read CA cert: %v", err)
	}

	// Create CA pool and add cert
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		log.Fatalf("Failed to add CA cert to pool")
	}

	// TLS config with your custom CA
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	httpClient := &http.Client{
		Transport: &transportWithAPIKey{
			apiKeyHeader: apiKeyHeader,
			rt: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		},
	}

	client, err := elastic.NewClient(
		elastic.SetURL(esURL),
		elastic.SetSniff(false),
		elastic.SetHttpClient(httpClient),
		//elastic.SetHealthcheck(false), // optional: disable startup healthcheck if self-signed certs
	)
	if err != nil {
		log.Fatalf("Error creating Elasticsearch client: %v", err)
	}
	// info, code, err := client.Ping(esURL).Do(context.Background())
	// if err != nil {
	// 	log.Fatalf("Ping failed: %v", err)
	// }
	// log.Printf("Elasticsearch returned with code %d and version %s", code, info.Version.Number)

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

func pushToElasticsearch(data []JoinedData) {
	// Connect to Elasticsearch
	esURL := "https://quickstart-es-http.default.svc:9200"
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
