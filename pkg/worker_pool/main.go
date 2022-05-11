/*
Package worker pool
Structure to facilitate with the worker pool pattern
https://gobyexample.com/worker-pools

Usage:

	type Task struct {
		i int
	}

	func (task Task) Run() {
		fmt.Printf("Processing task %d\n", task.i)
		time.Sleep(time.Duration(5) * time.Second)
		fmt.Printf("Processed task %d\n", task.i)
	}

	func main() {
		num_workers := 5
		num_tasks := 40
		pool := worker_pool.NewWorkerPool(num_workers, num_tasks)
		for i := 0; i < num_tasks; i++ {
			pool.AddTask(Task{i})
		}
		pool.Start()
		<-pool.Wait()
		fmt.Println("Worker pool done")
	}

`WorkerPool.Wait()` returns a signal that will block until all the tasks are completed
when you attempt to read it. The fact that it is a channel gives you the option to
listen to other channels that the tasks can write to at the same time:

	type Task struct {
		i               int
		message_channel chan string
	}

	func (task Task) Run() {
		task.message_channel <- fmt.Sprintf("Processing task %d", task.i)
		time.Sleep(time.Duration(5) * time.Second)
		task.message_channel <- fmt.Sprintf("Processed task %d", task.i)
	}

	func main() {
		num_workers := 5
		num_tasks := 40
		pool := worker_pool.NewWorkerPool(num_workers, num_tasks)
		message_channel := make(chan string)
		for i := 0; i < num_tasks; i++ {
			pool.AddTask(Task{i, message_channel})
		}
		pool.Start()
		exitfor := false
		for !exitfor {
			select {
			case msg := <- message_channel:
				fmt.Println(msg)
			case <-pool.Wait():
				exitfor = true
			}
		}
		fmt.Println("Worker pool done")
	}
*/

package worker_pool

import "sync"

type Task interface {
	Run(*WorkerPool)
}

type WorkerPool struct {
	numWorkers  int
	taskChannel chan Task
	wg          sync.WaitGroup
	IsAborted   bool
}

func NewWorkerPool(numWorkers, queueSize int) *WorkerPool {
	var pool WorkerPool
	pool.numWorkers = numWorkers
	pool.taskChannel = make(chan Task, queueSize)
	return &pool
}

func (pool *WorkerPool) AddTask(task Task) {
	pool.wg.Add(1)
	pool.taskChannel <- task
}

func (pool *WorkerPool) Start() {
	for i := 0; i < pool.numWorkers; i++ {
		go func() {
			for task := range pool.taskChannel {
				if !pool.IsAborted {
					task.Run(pool)
				}
				pool.wg.Done()
			}
		}()
	}
}

func (pool *WorkerPool) Abort() {
	// No need to protect this with a mutex because it will only go from false to true
	pool.IsAborted = true
}

func (pool *WorkerPool) Wait() <-chan struct{} {
	exitChannel := make(chan struct{})
	go func() {
		pool.wg.Wait()
		exitChannel <- struct{}{}
	}()
	return exitChannel
}
