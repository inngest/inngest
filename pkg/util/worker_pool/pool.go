package worker_pool

import (
	"sync"
)

type Job[Data any] struct {
	ID           int
	Data         Data
	batchResults chan Result[Data]
}

type Result[Data any] struct {
	JobID int
	Data  Data
	Err   error
}

type BatchRequest[Data any] struct {
	Jobs    []Job[Data]
	Results chan []Result[Data]
}

type WorkerPool[Data any] struct {
	numWorkers int
	jobs       chan Job[Data]
	scheduler  chan BatchRequest[Data]
	quit       chan struct{}
	wg         sync.WaitGroup
	process    func(job Job[Data], workerID int) Result[Data]
}

func NewWorkerPool[Data any](numWorkers int, process func(job Job[Data], workerID int) Result[Data]) *WorkerPool[Data] {
	return &WorkerPool[Data]{
		numWorkers: numWorkers,
		process:    process,
		jobs:       make(chan Job[Data]),
		scheduler:  make(chan BatchRequest[Data]),
		quit:       make(chan struct{}),
	}
}

func (wp *WorkerPool[Data]) Start() {
	// Launch fixed number of worker goroutines
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go func(workerID int) {
			defer wp.wg.Done()
			for {
				select {
				case job, ok := <-wp.jobs:
					if !ok {
						return
					}
					result := wp.process(job, workerID)
					job.batchResults <- result
				case <-wp.quit:
					return
				}
			}
		}(i)
	}

	// Launch scheduler goroutine
	go wp.runScheduler()
}

func (wp *WorkerPool[Data]) runScheduler() {
	for batch := range wp.scheduler {
		batchSize := len(batch.Jobs)
		if batchSize == 0 {
			batch.Results <- []Result[Data]{}
			continue
		}

		batchResultsChan := make(chan Result[Data], 0)

		// Create results slice for this batch
		batchResults := make([]Result[Data], batchSize)

		// Launch a single collector goroutine for this batch
		go func(results chan []Result[Data]) {
			// Collect all results for this batch
			for i := 0; i < batchSize; i++ {
				result := <-batchResultsChan
				batchResults[i] = result
			}
			close(batchResultsChan)
			// Send completed batch results
			results <- batchResults
		}(batch.Results)

		// Submit all jobs in this batch
		for _, job := range batch.Jobs {
			job.batchResults = batchResultsChan
			wp.jobs <- job
		}
	}
}

func (wp *WorkerPool[Data]) SubmitBatch(jobs []Job[Data]) []Result[Data] {
	resultsChan := make(chan []Result[Data], 1)
	wp.scheduler <- BatchRequest[Data]{
		Jobs:    jobs,
		Results: resultsChan,
	}
	return <-resultsChan
}

func (wp *WorkerPool[Data]) Stop() {
	close(wp.scheduler)
	close(wp.quit)
	wp.wg.Wait()
	close(wp.jobs)
}
