package main

import (
	"Go_Load_Balancer/dispatchers"
	"Go_Load_Balancer/logger"
	"Go_Load_Balancer/models"
	"io"
	"net/http"
	"time"
)

var logServer = logger.New("Go Server", logger.ColorGreen)

func main() {
	startServer()
}

func startServer() {
	quitChannel := make(chan bool)

	dispatcher := dispatchers.NewDispatcher(quitChannel)
	dispatcher.Start("spring-api-asg", 30*time.Second)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, dispatcher)
	})
	logServer.Println("HTTP server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logServer.Fatalf("HTTP server crashed: %v", err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request, dispatcher *dispatchers.Dispatcher) {
	logServer.Printf("Handling Request %+v", r)
	responseChannel := make(chan []byte)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	request := &models.Request{
		Payload:         body,
		ResponseChannel: responseChannel,
		URL:             r.URL.Path,
		Method:          r.Method,
	}

	go func() {
		dispatcher.HandleRequest(request)
	}()
	logServer.Printf("Waiting for response ....")
	response := <-responseChannel
	logServer.Printf("Response reseived  successfully")
	w.Write(response)

}
