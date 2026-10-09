package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

type PutRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Mode  string `json:"mode"`
}

func main() {
	op := flag.String("op", "get", "operation: put, get, or dump")
	replica := flag.String("replica", "http://localhost:8001", "replica URL")
	key := flag.String("key", "", "key")
	value := flag.String("value", "", "value for PUT")
	mode := flag.String("mode", "eventual", "consistency mode: eventual or strong")
	flag.Parse()

	start := time.Now()

	switch *op {
	case "put":
		if *key == "" {
			log.Fatal("-key is required")
		}
		req := PutRequest{Key: *key, Value: *value, Mode: *mode}
		b, _ := json.Marshal(req)
		resp, err := http.Post(*replica+"/put", "application/json", bytes.NewReader(b))
		if err != nil {
			log.Fatal(err)
		}
		printResponse(resp, start)

	case "get":
		if *key == "" {
			log.Fatal("-key is required")
		}
		endpoint := *replica + "/get?key=" + url.QueryEscape(*key) + "&mode=" + url.QueryEscape(*mode)
		resp, err := http.Get(endpoint)
		if err != nil {
			log.Fatal(err)
		}
		printResponse(resp, start)

	case "dump":
		resp, err := http.Get(*replica + "/dump")
		if err != nil {
			log.Fatal(err)
		}
		printResponse(resp, start)

	default:
		log.Fatal("unknown -op. use put, get, or dump")
	}
}

func printResponse(resp *http.Response, start time.Time) {
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, body, "", "  "); err == nil {
		body = pretty.Bytes()
	}
	fmt.Printf("HTTP status: %s\n", resp.Status)
	fmt.Printf("Client measured latency: %.3f ms\n", float64(time.Since(start).Microseconds())/1000.0)
	fmt.Println(string(body))
}
