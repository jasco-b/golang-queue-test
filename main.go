package main

import (
	"fmt"
	"io"
	"jasco-b/go-queue-test/lib"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	// just create queue
	lib.GetQueue()
	http.HandleFunc("/", handleRequest)
	err := http.ListenAndServe(":7000", nil)
	if err != nil {
		panic(err)
	}

}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		addQueue(w, r)
		fmt.Println(lib.GetQueue().List())
		return
	} else if r.Method == "GET" {
		getQueue(w, r)
		fmt.Println(lib.GetQueue().List())
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	io.WriteString(w, "")
}

func addQueue(w http.ResponseWriter, r *http.Request) {
	queueName := getQueueNameFromRequest(r)
	queueValue := r.URL.Query().Get("v")
	if queueName == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if queueValue == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	q := lib.GetQueue()
	q.Add(queueName, queueValue)

	io.WriteString(w, "this is put: queue:"+r.URL.Path+" value:"+queueValue+";")
}

func getQueue(w http.ResponseWriter, r *http.Request) {
	queueName := getQueueNameFromRequest(r)

	if queueName == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	timeoutQueryValue := strings.TrimSpace(r.URL.Query().Get("timeout"))
	timeout, err := strconv.Atoi(timeoutQueryValue)
	if timeoutQueryValue != "" && err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dTimeout := time.Duration(timeout)
	queueValue := lib.GetQueue().Get(queueName)
	if queueValue == "" && timeout == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if queueValue != "" {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, queueValue)
		return
	}

	queueChannel := lib.GetListener().Subscribe(queueName)
	for {
		select {
		case value := <-queueChannel:
			lib.GetListener().Unsubscribe(queueName)
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, value)
			return
		case <-time.After(dTimeout * time.Second):
			lib.GetListener().Unsubscribe(queueName)
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

}

func getQueueNameFromRequest(r *http.Request) string {
	if len(r.URL.Path) > 0 && r.URL.Path[:1] == "/" {
		return r.URL.Path[1:]
	}
	return r.URL.Path
}
