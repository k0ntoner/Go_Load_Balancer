package workers

import (
	"EntropyLoadBalancer/logger"
	"EntropyLoadBalancer/models"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

var logWorker = logger.New("Worker", logger.ColorBlue)

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
		IdleConnTimeout:     90 * time.Second,
	},
	Timeout: 30 * time.Second,
}

type Worker struct {
	ID             int
	TargetInstance *models.Instance
	QuitChannel    chan bool
	Retry          byte
}

func NewWorker(id int, targetInstance *models.Instance, QuitChannel chan bool) *Worker {
	return &Worker{
		ID:             id,
		TargetInstance: targetInstance,
		QuitChannel:    QuitChannel,
		Retry:          3,
	}
}

func (w *Worker) HandleRequest(request *models.Request) {
	w.handleRequest(request, 0)
}

func (w *Worker) handleRequest(request *models.Request, numberOfTry byte) {
	start := time.Now()
	targetURL := fmt.Sprintf("http://%s:8080%s", w.TargetInstance.IPAddress, request.URL)

	logWorker.Printf("[START] Worker #%d handling request ID: %s, Method: %s, URL: %s → Target: %s",
		w.ID, request.ID, request.Method, request.URL, targetURL)

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
			logWorker.Printf("[FAIL] Worker #%d: Instance [%s] unreachable. Error: %v", w.ID, w.TargetInstance.IPAddress, err)
			request.ResponseChannel <- []byte("503 Service Unavailable")
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
		w.ID, request.ID, w.TargetInstance.IPAddress, response.StatusCode, time.Since(start))

	request.ResponseChannel <- body
	logWorker.Printf("Repsonse was delivered successfully")

}
