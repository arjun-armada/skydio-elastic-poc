package main

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
)

func main() {
	username := "elastic"
	password := ""
	url := "https://localhost:9200"
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "Basic "+auth)

	client := http.Client{
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
