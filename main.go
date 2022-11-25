package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	ctx, done := context.WithCancel(context.Background())
	g, gctx := errgroup.WithContext(ctx)

	// goroutine to check for signals to gracefully finish all functions
	g.Go(func() error {
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

		select {
		case sig := <-signalChannel:
			fmt.Printf("Received signal: %s\n", sig)
			done()
		case <-gctx.Done():
			fmt.Printf("closing signal goroutine\n")
			return gctx.Err()
		}

		return nil
	})

	// ticker every 1s for making request to db1
	g.Go(func() error {
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ticker.C:
				err := makeRequest("3323", "db1")
				if err != nil {
					fmt.Println("error:db1", err)
				}
			case <-gctx.Done():
				fmt.Printf("closing ticker 1s goroutine\n")
				return gctx.Err()
			}
		}
	})

	// ticker every 1s for making request to db2
	g.Go(func() error {
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ticker.C:
				err := makeRequest("3323", "db2")
				if err != nil {
					fmt.Println("error:db2", err)
				}
			case <-gctx.Done():
				fmt.Printf("closing ticker 1s goroutine\n")
				return gctx.Err()
			}
		}
	})

	// ticker every 1s for making request to db3
	g.Go(func() error {
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ticker.C:
				err := makeRequest("3323", "db3")
				if err != nil {
					fmt.Println("error:db3", err)
				}
			case <-gctx.Done():
				fmt.Printf("closing ticker 1s goroutine\n")
				return gctx.Err()
			}
		}
	})

	// wait for all errgroup goroutines
	if err := g.Wait(); err == nil || err == context.Canceled {
		fmt.Println("finished clean")
	} else {
		fmt.Printf("received error: %v", err)
	}
}

func makeRequest(port, db string) error {
	// login
	baseURL := fmt.Sprintf("http://127.0.0.1:%s", port)
	jsonBody := []byte(`{"user": "aW1tdWRi", "password": "aW1tdWRi"}`)
	bodyReader := bytes.NewReader(jsonBody)
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/%s", baseURL, "login"), bodyReader)
	if err != nil {
		fmt.Printf("client: could not create request: %s\n", err)
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	dec := make(map[string]string)
	err = json.Unmarshal(body, &dec)
	if err != nil {
		return err
	}
	loginToken := dec["token"]

	// select database
	url := fmt.Sprintf("%s/db/use/%s", baseURL, db)
	// url := fmt.Sprintf("http://127.0.0.1:3323/db/use/%s", db)
	req, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Printf("client: could not select db: %s\n", err)
		return err
	}
	req.Header.Add("Authorization", loginToken)
	req.Header.Add("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		return err
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	dec = make(map[string]string)
	err = json.Unmarshal(body, &dec)
	if err != nil {
		return err
	}
	dbToken := dec["token"]
	if dbToken == "" {
		return errors.New("database token not found")
	}

	// verified set
	data := fmt.Sprintf("data%d", rand.Intn(1000000))
	dataEnc := base64.StdEncoding.EncodeToString([]byte(data))
	url = fmt.Sprintf("%s/db/%s/verified/set", baseURL, db)
	// url = fmt.Sprintf("http://127.0.0.1:3323/db/verified/set")
	jsonBody = []byte(fmt.Sprintf(`{
		"setRequest": {
		  "KVs": [
			{
			"key": "%s",
			"value": "%s"
			}
		  ]
		}
	  }`, dataEnc, dataEnc))
	bodyReader = bytes.NewReader(jsonBody)
	req, err = http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		fmt.Printf("client: could not select db: %s\n", err)
		return err
	}
	req.Header.Add("Authorization", dbToken)
	req.Header.Add("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		return err
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	dresp := make(map[string]interface{})
	err = json.Unmarshal(body, &dresp)
	if err != nil {
		return err
	}
	if _, ok := dresp["id"]; !ok {
		return errors.New(fmt.Sprintf("set:%s", dresp))
	}

	// verified get
	url = fmt.Sprintf("%s/db/%s/verified/get", baseURL, db)
	// url = fmt.Sprintf("http://127.0.0.1:3323/db/verified/get")
	jsonBody = []byte(fmt.Sprintf(`{
		"keyRequest": {
		  "key": "%s"
		}
	  }`, dataEnc))
	bodyReader = bytes.NewReader(jsonBody)
	req, err = http.NewRequest(http.MethodPost, url, bodyReader)
	if err != nil {
		fmt.Printf("client: could not select db: %s\n", err)
		return err
	}
	req.Header.Add("Authorization", dbToken)
	req.Header.Add("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("client: error making http request: %s\n", err)
		return err
	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	dresp = make(map[string]interface{})
	err = json.Unmarshal(body, &dresp)
	if err != nil {
		return err
	}
	if _, ok := dresp["tx"]; !ok {
		return errors.New(fmt.Sprintf("get:%s", dresp))
	}

	log.Println("data: ", db, data)
	return nil
}
