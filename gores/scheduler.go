package gores

import (
	"log"
)

// Scheduler represents a scheduler that schedule delayed and failed jobs in Gores
type Scheduler struct {
	resq          *ResQ
	timestampChan chan int64
	shutdownChan  chan bool
}

// NewScheduler initializes a new shceduler
func NewScheduler(config *Config) *Scheduler {
	var sche *Scheduler
	resq := NewResQ(config)
	if resq == nil {
		log.Fatalf("ERROR Initializing ResQ(), cannot initialize Scheduler")
		return nil
	}
	sche = &Scheduler{
		resq:          resq,
		timestampChan: make(chan int64, 1),
		shutdownChan:  make(chan bool, 1),
	}
	return sche
}

// ScheduleShutdown schedules the shutdown of sheduler
func (sche *Scheduler) ScheduleShutdown() {
	sche.shutdownChan <- true
}

// NextDelayedTimestamps fetches delayed jobs from Redis and place them into channel
func (sche *Scheduler) NextDelayedTimestamps() {
	for {
		timestamp := sche.resq.NextDelayedTimestamp()
		log.Printf("timestamp of delayed item: %d\n", timestamp)
		if timestamp != 0 {
			sche.timestampChan <- timestamp
		} else {
			/* breaks when there is no delayed items in the 'resq:delayed:timestamp' queue*/
			break
		}
	}
	sche.ScheduleShutdown()
}

// HandleDelayedItems re-enqueue delayed or failed jobs back to redis
func (sche *Scheduler) HandleDelayedItems() {
	go sche.NextDelayedTimestamps()
	for {
		select {
		case timestamp := <-sche.timestampChan:
			item := sche.resq.NextItemForTimestamp(timestamp)
			if item != nil {
				log.Println(item)
				err := sche.resq.Enqueue(item)
				if err != nil {
					log.Fatalf("ERROR Enqueue Delayed Item: %s", err)
				}
			}
		case <-sche.shutdownChan:
			log.Println("Finish Handling Delayed Items")
			return
		}
	}
}

// Run startups the scheduler
func (sche *Scheduler) Run() {
	sche.HandleDelayedItems()
}
