package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	endpoint          = "http://localhost:8003/group03/api/books"
	startPointWorkers = 1
	maxWorkers        = 100
)

// returns time taken in milliseconds
func makeRequest(client *http.Client, url string) (int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	startTime := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	elapsed := time.Since(startTime)

	return int(elapsed.Milliseconds()), nil
}

type WorkerData struct {
	mutex            sync.Mutex
	totalTime        int
	connectedWorkers int
}

func worker(wg *sync.WaitGroup, workerData *WorkerData) {
	client := &http.Client{}

	defer wg.Done()
	timeTaken, err := makeRequest(client, endpoint)
	if err != nil {
		fmt.Println("Error in worker:", err)
		return
	}

	workerData.mutex.Lock()
	workerData.totalTime += timeTaken
	workerData.connectedWorkers++
	workerData.mutex.Unlock()
}

// returns time taken in milliseconds and the number of workers that were connected
func getAverageResponseTime(amountOfWorkers int) (int, int, error) {
	wg := sync.WaitGroup{}
	workerData := WorkerData{
		mutex:            sync.Mutex{},
		totalTime:        0,
		connectedWorkers: 0,
	}

	for i := 0; i < amountOfWorkers; i++ {
		wg.Add(1)
		go worker(&wg, &workerData)
	}

	wg.Wait()

	if workerData.connectedWorkers == 0 {
		return 0, 0, fmt.Errorf("no workers connected")
	}

	return workerData.totalTime / workerData.connectedWorkers, workerData.connectedWorkers, nil
}

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}

type Response struct {
	XAxis         []int `json:"xAxis"`
	ResponseTimes []int `json:"responseTimes"`
	Denied        []int `json:"denied"`
	Done          bool  `json:"done"`
}

func main() {
	failedConnections := make([]int, 0)
	responseTimes := make([]int, 0)

	go func() {
		for workers := startPointWorkers; workers <= maxWorkers; workers++ {
			timeTaken, connectedWorkers, err := getAverageResponseTime(workers)
			if err != nil {
				fmt.Println("Error in getAverageResponseTime:", err)
			}

			failedConnections = append(failedConnections, workers-connectedWorkers)
			responseTimes = append(responseTimes, timeTaken)
		}
		fmt.Println("Done making requests.")
	}()

	fs := http.FileServer(http.Dir("public"))

	http.Handle("/", fs)

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		response := Response{
			XAxis:         makeRange(startPointWorkers, maxWorkers),
			ResponseTimes: responseTimes,
			Denied:        failedConnections,
			Done:          len(responseTimes) == maxWorkers,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
	})

	fmt.Println("Server started on http://localhost:8081")
	http.ListenAndServe(":8081", nil)
}
