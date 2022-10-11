/*
Package worker pool
Structure to facilitate with the worker pool pattern
https://gobyexample.com/worker-pools

Usage:

	type Task struct {
		i int
	}

	func (task *Task) Run(send func(string), abort funct()) {
		send(fmt.Sprintf("Processing task %d\n", task.i))
		time.Sleep(time.Duration(5) * time.Second)
		send(fmt.Sprintf("Processed task %d\n", task.i))
	}

	func main() {
		numWorkers := 5
		numTasks := 40
		pool := worker_pool.New(numWorkers, numTasks)
		for i := 0; i < numTasks; i++ {
			pool.Add(&Task{i})
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
		numWorkers := 5
		numTasks := 40
		resultChannel := make(chan int)
		pool := worker_pool.NewWorkerPool(numWorkers, numTasks)
		for i := 0; i < numTasks; i++ {
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
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gosuri/uilive"
	"github.com/mattn/go-isatty"
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
	numWorkers       int
	numTasks         int
	taskChannel      chan taskContainer_t
	innerWaitGroup   sync.WaitGroup
	outerWaitGroup   sync.WaitGroup
	counter          int
	forceNotTerminal bool

	IsAborted bool
}

func New(numWorkers, numTasks int, forceNotTerminal bool) *Pool {
	var pool Pool
	pool.numWorkers = numWorkers
	pool.numTasks = numTasks
	pool.taskChannel = make(chan taskContainer_t, numTasks)
	pool.forceNotTerminal = forceNotTerminal
	return &pool
}

func (pool *Pool) Add(task Task) {
	pool.innerWaitGroup.Add(1)
	pool.taskChannel <- taskContainer_t{pool.counter, task}
	pool.counter += 1
}

func (pool *Pool) Start() {
	messages := make([]string, pool.numTasks+1)
	messageChannel := make(chan message_t)
	writer := uilive.New()
	if !pool.forceNotTerminal && isatty.IsTerminal(os.Stdout.Fd()) {
		writer.Start()
	}
	pool.outerWaitGroup.Add(1)

	var finishedTasks int32 = 0

	for i := 0; i < pool.numWorkers; i++ {
		go func() {
			for taskContainer := range pool.taskChannel {
				if !pool.IsAborted {
					send := func(body string) {
						messageChannel <- message_t{taskContainer.i, body}
					}
					taskContainer.task.Run(send, pool.abort)
				}
				if !pool.forceNotTerminal && isatty.IsTerminal(os.Stdout.Fd()) {
					atomic.AddInt32(&finishedTasks, 1)
					messageChannel <- message_t{
						pool.numTasks,
						makeProgressBar(finishedTasks, pool.numTasks),
					}
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

	printMessages := func() {
		var tmpMessages []string
		for _, line := range messages {
			if len(line) > 0 {
				tmpMessages = append(tmpMessages, line)
			}
		}
		fmt.Fprintln(writer, strings.Join(tmpMessages, "\n"))
		writer.Flush()
	}

	go func() {
		exitfor := false
		for !exitfor {
			select {
			case msg := <-messageChannel:
				if !pool.forceNotTerminal && isatty.IsTerminal(os.Stdout.Fd()) {
					messages[msg.i] = msg.body
					printMessages()
				} else {
					fmt.Println(msg.body)
				}
			case <-waitChannel:
				exitfor = true
				if !pool.forceNotTerminal && isatty.IsTerminal(os.Stdout.Fd()) {
					writer.Stop()
				}
				close(messageChannel)
				pool.outerWaitGroup.Done()
			}
		}
	}()

	if !pool.forceNotTerminal && isatty.IsTerminal(os.Stdout.Fd()) {
		messageChannel <- message_t{
			pool.numTasks,
			makeProgressBar(finishedTasks, pool.numTasks),
		}
	}
}

func (pool *Pool) abort() {
	// No need to protect this with a Mutex since it only goes from false -> true
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

func makeProgressBar(low int32, high int) string {
	length := 30
	dots := length * int(low) / high
	return fmt.Sprintf(
		"[%s%s] (%d / %d)",
		strings.Repeat("#", dots),
		strings.Repeat("-", length-dots),
		low,
		high,
	)
}
