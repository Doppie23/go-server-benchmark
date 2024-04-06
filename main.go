package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/akamensky/argparse"
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

func worker(endpoint string, wg *sync.WaitGroup, workerData *WorkerData) {
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
func getAverageResponseTime(endpoint string, amountOfWorkers int) (int, int, error) {
	wg := sync.WaitGroup{}
	workerData := WorkerData{
		mutex:            sync.Mutex{},
		totalTime:        0,
		connectedWorkers: 0,
	}

	for i := 0; i < amountOfWorkers; i++ {
		wg.Add(1)
		go worker(endpoint, &wg, &workerData)
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

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

type Response struct {
	XAxis         []int `json:"xAxis"`
	ResponseTimes []int `json:"responseTimes"`
	Denied        []int `json:"denied"`
	Done          bool  `json:"done"`
}

func main() {
	parser := argparse.NewParser("go-server-benchmark", "A simple benchmark to test the performance of a server. By automatically making requests and scaling up the amount of requests at once to the server and measuring the response time.")

	endpoint := parser.String("e", "endpoint", &argparse.Options{Required: true, Help: "The endpoint to test."})
	port := parser.Int("p", "port", &argparse.Options{Default: 8081, Help: "The port the webserver is running on."})
	startPointWorkers := parser.Int("s", "start-workers", &argparse.Options{Default: 1, Help: "The amount of workers to start with."})
	maxWorkers := parser.Int("m", "max-workers", &argparse.Options{Default: 100, Help: "The maximum amount of workers to test."})

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	failedConnections := make([]int, 0)
	responseTimes := make([]int, 0)

	doneMakingRequests := false

	go func() {
		for workers := *startPointWorkers; workers <= *maxWorkers; workers++ {
			timeTaken, connectedWorkers, err := getAverageResponseTime(*endpoint, workers)
			if err != nil {
				fmt.Println("Error in getAverageResponseTime:", err)
			}

			failedConnections = append(failedConnections, workers-connectedWorkers)
			responseTimes = append(responseTimes, timeTaken)
		}
		doneMakingRequests = true
		fmt.Println("Done making requests.")
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(getHtml()))
	})

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		response := Response{
			XAxis:         makeRange(*startPointWorkers, *maxWorkers),
			ResponseTimes: responseTimes,
			Denied:        failedConnections,
			Done:          doneMakingRequests,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonResponse)
	})

	url := fmt.Sprintf("http://localhost:%d", *port)

	fmt.Printf("Server started on %s\n", url)
	fmt.Println("Opening browser...")
	err = openBrowser(url)
	if err != nil {
		fmt.Println("Error opening browser:", err)
	}
	http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
}
