package workers

import (
	"Go_Load_Balancer/logger"
	"Go_Load_Balancer/models"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

var logWorker = logger.New("Worker", logger.ColorBlue)

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100000,
		MaxIdleConnsPerHost: 10000,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 30 * time.Second,
}

type Worker struct {
	ID                     int
	TargetInstance         *models.Instance
	QuitChannel            chan bool
	Retry                  byte
	NumberOfActiveRequests int32
	DeadChannel            chan string
}

func NewWorker(id int, targetInstance *models.Instance, QuitChannel chan bool, DeadChannel chan string) *Worker {
	return &Worker{
		ID:                     id,
		TargetInstance:         targetInstance,
		QuitChannel:            QuitChannel,
		Retry:                  10,
		NumberOfActiveRequests: 0,
		DeadChannel:            DeadChannel,
	}
}

func (w *Worker) Increment() {
	atomic.AddInt32(&w.NumberOfActiveRequests, 1)
}

func (w *Worker) Decrement() {
	atomic.AddInt32(&w.NumberOfActiveRequests, -1)
}

func (w *Worker) GetActiveRequests() int32 {
	return atomic.LoadInt32(&w.NumberOfActiveRequests)
}

func (w *Worker) HandleRequest(request *models.Request) {
	w.Increment()
	w.handleRequest(request, 0)
}

func (w *Worker) handleRequest(request *models.Request, numberOfTry byte) {
	defer w.Decrement()
	start := time.Now()
	targetURL := fmt.Sprintf("http://%s:8080%s", w.TargetInstance.IPAddress, request.URL)

	logWorker.Printf("[START] Worker #%d handling request ID: %s, Method: %s, URL: %s → Target: %s",
		w.ID, request.Method, request.URL, targetURL)

	var response *http.Response
	var err error

	req, err := http.NewRequest(request.Method, targetURL, bytes.NewBuffer(request.Payload))

	if err != nil {
		logWorker.Printf("[ERROR] Worker #%d: Failed to create request: %v", w.ID, err)
		request.ResponseChannel <- []byte("400 Bad Request")
		return
	}

	req.Header.Set("Content-Type", "application/json")

	response, err = httpClient.Do(req)

	if err != nil || response == nil {

		if numberOfTry < w.Retry {
			w.handleRequest(request, numberOfTry+1)
		} else {
			logWorker.Printf("[FAIL] Worker #%d: Instance [%s] unreachable", w.ID, w.TargetInstance.IPAddress)
			w.DeadChannel <- w.TargetInstance.ID
			request.ResponseChannel <- []byte("503 Service Unavailable")
			return
		}
		return
	}

	defer response.Body.Close()

	w.TargetInstance.CountOfLoads++
	w.TargetInstance.LastUsedTime = time.Now()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		logWorker.Printf("[ERROR] Worker #%d: Failed to read response from [%s]. Error: %v",
			w.ID, w.TargetInstance.IPAddress, err)
		request.ResponseChannel <- []byte("500 Internal Error")
		return
	}

	logWorker.Printf("[SUCCESS] Worker #%d: Request ID %s to [%s] completed. Status Code: %d, Time: %s",
		w.ID, w.TargetInstance.IPAddress, response.StatusCode, time.Since(start))

	request.ResponseChannel <- body
	logWorker.Printf("Repsonse was delivered successfully")

}
