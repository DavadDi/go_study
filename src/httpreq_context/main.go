package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func HttpDoTest(ctx context.Context, resChan chan<- string) error {
	start := time.Now()

	repoUrl := "https://api.github.com/repos/campoy/golang-plugins"
	req, err := http.NewRequest("GET", repoUrl, nil)
	if err != nil {
		return fmt.Errorf("http.NewRequest Error: %s", err.Error())
	}

	// in go >= 1.7
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("client.Do Error: %s", err.Error())
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ioutil.ReadAll Error: %s", err.Error())
	}

	log.Printf("Read body size [%d]", len(data))
	log.Println("CostTime is: " + time.Since(start).String())

	resChan <- string(data)

	return nil
}

func main() {
	deadline := 1
	d := time.Now().Add(time.Duration(deadline) * time.Second) // deadline max
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	resChan := make(chan string)

	go HttpDoTest(ctx, resChan)

	var resData string
	select {
	case <-ctx.Done():
		fmt.Println(ctx.Err())

		/* just for ex use. No used*/
	case <-time.Tick(time.Duration(time.Duration(deadline*2) * time.Second)):
		fmt.Println("Time over!")

	case resData = <-resChan:
		fmt.Println("Read data finished")
	}

	log.Printf("Read data size: [%d]", len(resData))
}
