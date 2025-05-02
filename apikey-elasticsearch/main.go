package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
)

func main() {
	apiKey := ""
	url := "https://localhost:9200/"

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "ApiKey "+apiKey)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // dev only
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))

}
