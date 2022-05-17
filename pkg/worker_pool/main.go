/*
Package worker pool
Structure to facilitate with the worker pool pattern
https://gobyexample.com/worker-pools

Usage:

	type Task struct {
		i int
	}

	func (task Task) Run(send func(string), abort funct()) {
		send(fmt.Sprintf("Processing task %d\n", task.i))
		time.Sleep(time.Duration(5) * time.Second)
		send(fmt.Sprintf("Processed task %d\n", task.i))
	}

	func main() {
		num_workers := 5
		num_tasks := 40
		pool := worker_pool.New(num_workers, num_tasks)
		for i := 0; i < num_tasks; i++ {
			pool.Add(Task{i})
		}
		pool.Start()
		<-pool.Wait()
		fmt.Println("Worker pool done")
	}


Each task gets a portion of an output that gets updated while the workers are
running (using [uilive](https://github.com/gosuri/uilive)). Each invocation of
'send' will replace the portion of the output dedicated to the task.

`WorkerPool.Wait()` returns a channel that will block until all the tasks are completed
when you attempt to read it. The fact that it is a channel gives you the option to
listen to other channels that the tasks can write to at the same time:

	type Task struct {
		i             int
		resultChannel chan int
	}

	func (task Task) Run() {
		time.Sleep(time.Duration(5) * time.Second)
		resultChannel <- task.i * task.i
	}

	func main() {
		num_workers := 5
		num_tasks := 40
		resultChannel := make(chan int)
		pool := worker_pool.NewWorkerPool(num_workers, num_tasks)
		for i := 0; i < num_tasks; i++ {
			pool.AddTask(Task{i, resultChannel})
		}
		pool.Start()
		waitChannel := pool.Wait()
		exitfor := false
		for !exitfor {
			select {
			case result := <- resultChannel:
				fmt.Printf("%d\n", result)
			case <- waitChannel:
				exitfor = true
			}
		}
		fmt.Println("Worker pool done")
	}

Calling 'abort' will make sure the workers will not pick up any new tasks.
However, tasks that are already in progress will continue. After the pool is
done, you can check the IsAborted field to see if any of the tasks aborted.

	type Task struct {
		i int
	}

	func (task Task) Run(send func(string), abort func()) {
		if task.i == 20 {
			abort()
			return
		}
		// Do stuff
	}

	func main() {
		pool := worker_pool.New(5, 40)
		for i := 0; i < 40; i++ {
			pool.Add(Task{i})
		}
		pool.Start()
		<-pool.Wait()
		if pool.IsAborted {
			fmt.Pritnln("Something went wrong")
		}
	}
*/

package worker_pool

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gosuri/uilive"
)

type Task interface {
	Run(send func(string), abort func())
}

type taskContainer_t struct {
	i    int
	task Task
}

type message_t struct {
	i    int
	body string
}

type Pool struct {
	numWorkers     int
	taskChannel    chan taskContainer_t
	innerWaitGroup sync.WaitGroup
	outerWaitGroup sync.WaitGroup
	counter        int
	messages       []string
	messageChannel chan message_t
	writer         *uilive.Writer

	IsAborted bool
}

func New(numWorkers, numTasks int) *Pool {
	var pool Pool
	pool.numWorkers = numWorkers
	pool.taskChannel = make(chan taskContainer_t, numTasks)
	pool.messages = make([]string, numTasks)
	pool.messageChannel = make(chan message_t)
	return &pool
}

func (pool *Pool) Add(task Task) {
	pool.innerWaitGroup.Add(1)
	pool.taskChannel <- taskContainer_t{pool.counter, task}
	pool.counter += 1
}

func (pool *Pool) Start() {
	pool.writer = uilive.New()
	pool.writer.Start()
	pool.outerWaitGroup.Add(1)

	for i := 0; i < pool.numWorkers; i++ {
		go func() {
			for taskContainer := range pool.taskChannel {
				if !pool.IsAborted {
					send := func(body string) {
						pool.messageChannel <- message_t{taskContainer.i, body}
					}
					taskContainer.task.Run(send, pool.abort)
				}
				pool.innerWaitGroup.Done()
			}
		}()
	}

	waitChannel := make(chan struct{})
	go func() {
		pool.innerWaitGroup.Wait()
		waitChannel <- struct{}{}
	}()

	go func() {
		exitfor := false
		for !exitfor {
			select {
			case msg := <-pool.messageChannel:
				pool.messages[msg.i] = msg.body
				var tmpMessages []string
				for _, line := range pool.messages {
					if len(line) > 0 {
						tmpMessages = append(tmpMessages, line)
					}
				}
				fmt.Fprintln(pool.writer, strings.Join(tmpMessages, "\n"))
				pool.writer.Flush()
			case <-waitChannel:
				exitfor = true
				pool.writer.Stop()
				pool.outerWaitGroup.Done()
			}
		}
	}()
}

func (pool *Pool) abort() {
	pool.IsAborted = true
}

func (pool *Pool) Wait() <-chan struct{} {
	waitChannel := make(chan struct{})
	go func() {
		pool.outerWaitGroup.Wait()
		waitChannel <- struct{}{}
	}()
	return waitChannel
}
