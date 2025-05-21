package dispatchers

import (
	"EntropyLoadBalancer/logger"
	"EntropyLoadBalancer/models"
	"EntropyLoadBalancer/workers"
	"log"
	"math"
	"sync"
)

var logDispatcher = logger.New("Dispatcher", logger.ColorYellow)

type Dispatcher struct {
	WorkerList   []*workers.Worker
	InstanceList []*models.Instance
	QuitChannel  chan bool
	StartChannel chan bool
	Size         uint64
	mu           sync.RWMutex
}

func NewDispatcher(instanceList []*models.Instance, quitChannel chan bool) *Dispatcher {
	dispatcher := &Dispatcher{
		InstanceList: instanceList,
		WorkerList:   make([]*workers.Worker, len(instanceList)),
		QuitChannel:  quitChannel,
		StartChannel: make(chan bool),
		Size:         uint64(len(instanceList)),
	}
	return dispatcher
}

func (d *Dispatcher) Start() {
	for i := 0; i < len(d.InstanceList); i++ {
		go func(index int) {
			quitChannel := make(chan bool)
			instance := d.InstanceList[index]
			createdWorker := workers.NewWorker(index+1, instance, quitChannel)
			d.WorkerList[index] = createdWorker
			logDispatcher.Printf("Created worker #%v.\n", createdWorker)

			<-quitChannel
			log.Fatalf("Worker #%d Instance #%v crashed.\n", createdWorker.ID, createdWorker.TargetInstance)
			log.Fatalf("Removing worker #%v.....\n", createdWorker.ID)
			d.removeDependencies(createdWorker, instance)
			log.Fatalf("Worker #%v has been removed.\n", createdWorker.ID)

			if d.Size == 0 {
				log.Fatalf("Every worker was destroyed, dispatcher doesn't available")
				d.QuitChannel <- true
				return
			}
		}(i)
	}

}

func (d *Dispatcher) HandleRequest(request *models.Request) {

	worker := d.getWorker()
	logDispatcher.Printf("Worker [#%d] handle request #%v", worker.ID, request)
	worker.HandleRequest(request)
}

func (d Dispatcher) getWorker() *workers.Worker {
	if len(d.WorkerList) == 0 {
		return nil
	}

	var selected *workers.Worker
	minRequests := int32(math.MaxInt32)

	for _, worker := range d.WorkerList {
		active := worker.GetActiveRequests()
		if active < minRequests {
			minRequests = active
			selected = worker
		}
	}

	return selected
}

func removeWorker(list []*workers.Worker, target *workers.Worker) []*workers.Worker {
	for i, w := range list {
		if w == target {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func removeInstance(list []*models.Instance, target *models.Instance) []*models.Instance {
	for i, s := range list {
		if s == target {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func (d *Dispatcher) removeDependencies(w *workers.Worker, instance *models.Instance) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.WorkerList = removeWorker(d.WorkerList, w)
	d.Size--
	d.InstanceList = removeInstance(d.InstanceList, instance)
}
