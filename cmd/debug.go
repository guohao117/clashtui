//go:build debug

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	clashHost = "http://127.0.0.1:9090"
	authToken = "your-secret-here" // 替换为你的 token
)

type ClashLog struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Time    string `json:"time"`
}

func main() {
	fmt.Println("Starting debug...")

	client := &http.Client{}
	req, err := http.NewRequest("GET", clashHost+"/logs", nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	if authToken != "" {
		req.Header.Add("Authorization", "Bearer "+authToken)
	}

	fmt.Println("Sending request...")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Response status: %s\n", resp.Status)
	fmt.Println("Reading logs...")

	decoder := json.NewDecoder(resp.Body)
	count := 0
	for {
		var log ClashLog
		err := decoder.Decode(&log)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error decoding log: %v\n", err)
			return
		}
		count++
		fmt.Printf("Log %d: %+v\n", count, log)

		// 每读取一条日志暂停一下，方便观察
		time.Sleep(time.Millisecond * 500)
	}

	fmt.Printf("Total logs read: %d\n", count)
}
