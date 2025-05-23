package dispatchers

import (
	"Go_Load_Balancer/config"
	"Go_Load_Balancer/logger"
	"Go_Load_Balancer/models"
	"Go_Load_Balancer/workers"
	"math"
	"sync"
	"time"
)

var logDispatcher = logger.New("Dispatcher", logger.ColorYellow)

type Dispatcher struct {
	WorkerList   []*workers.Worker
	InstanceList []*models.Instance
	QuitChannel  chan bool
	StartChannel chan bool
	Size         uint64
	mu           sync.RWMutex
	DeadChannel  chan string
}

func NewDispatcher(quitChannel chan bool) *Dispatcher {
	dispatcher := &Dispatcher{
		InstanceList: []*models.Instance{},
		WorkerList:   []*workers.Worker{},
		QuitChannel:  quitChannel,
		StartChannel: make(chan bool),
		DeadChannel:  make(chan string),
	}
	return dispatcher
}

func (d *Dispatcher) Start(autoScalingGroupName string, region string, refreshInterval time.Duration) {
	go func() {
		for deadID := range d.DeadChannel {
			d.mu.Lock()
			d.removeByID(deadID)
			d.mu.Unlock()
			logDispatcher.Printf("[DISPATCHER] removed dead instance %s", deadID)
		}
	}()

	d.refresh(autoScalingGroupName, region)

	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			d.refresh(autoScalingGroupName, region)
		}
	}()
}

func (d *Dispatcher) refresh(autoScalingGroupName string, region string) {
	logDispatcher.Printf("Refreshing instances from ASG %q…", autoScalingGroupName)
	newList, err := config.GetInstances(autoScalingGroupName, region)
	if err != nil {
		logDispatcher.Printf("[ERROR] fetch instances: %v", err)
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	oldMap := make(map[string]*models.Instance, len(d.InstanceList))
	for _, inst := range d.InstanceList {
		oldMap[inst.ID] = inst
	}
	newMap := make(map[string]*models.Instance, len(newList))
	for _, inst := range newList {
		newMap[inst.ID] = inst
	}

	for id, inst := range newMap {
		if _, exists := oldMap[id]; !exists {
			logDispatcher.Printf("→ Adding new instance %s", id)
			d.InstanceList = append(d.InstanceList, inst)

			quitCh := make(chan bool)
			w := workers.NewWorker(len(d.WorkerList)+1, inst, quitCh, d.DeadChannel)
			d.WorkerList = append(d.WorkerList, w)
			go func(worker *workers.Worker, ch chan bool) {
				<-ch
				logDispatcher.Printf("Worker %d quit", worker.ID)
			}(w, quitCh)
		}
	}

	for i := 0; i < len(d.InstanceList); {
		inst := d.InstanceList[i]
		if _, still := newMap[inst.ID]; !still {
			logDispatcher.Printf("← Removing instance %s", inst.ID)
			for j, w := range d.WorkerList {
				if w.TargetInstance.ID == inst.ID {
					w.QuitChannel <- true
					d.WorkerList = append(d.WorkerList[:j], d.WorkerList[j+1:]...)
					break
				}
			}
			d.InstanceList = append(d.InstanceList[:i], d.InstanceList[i+1:]...)
			continue
		}
		i++
	}
}

func (d *Dispatcher) removeByID(id string) {
	for i, w := range d.WorkerList {
		if w.TargetInstance.ID == id {
			w.QuitChannel <- true
			d.WorkerList = append(d.WorkerList[:i], d.WorkerList[i+1:]...)
			break
		}
	}
	for i, inst := range d.InstanceList {
		if inst.ID == id {
			d.InstanceList = append(d.InstanceList[:i], d.InstanceList[i+1:]...)
			break
		}
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
