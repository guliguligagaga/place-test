package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	//"runtime"
	"strings"
	"sync"
)

func main() {
	url := "http://localhost:5001/draw"
	numWorkers := runtime.NumCPU()
	//numWorkers := 2
	var wg sync.WaitGroup
	tasks := make(chan [2]int, 500*500)

	worker := func(tasks <-chan [2]int, wg *sync.WaitGroup) {
		defer wg.Done()
		for task := range tasks {
			i, j := task[0], task[1]
			n := rand.Int31n(33)
			payload := strings.NewReader(fmt.Sprintf("{\"x\":%d,\"y\":%d,\"color\":%d, \"timestamp\":%d}", i, j, n, time.Now().UnixMilli()))
			req, _ := http.NewRequest("POST", url, payload)
			req.Header.Add("Content-Type", "application/json")
			_, _ = http.DefaultClient.Do(req)
			println("sent", time.Now().Unix())
		}
	}

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go worker(tasks, &wg)
	}

	for i := 0; i < 500; i++ {
		for j := 0; j < 500; j++ {
			tasks <- [2]int{i, j}
		}
	}
	close(tasks)
	wg.Wait()
}
