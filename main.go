package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var queueList map[string]IQueue
var listener IListener

func main() {
	queueList = make(map[string]IQueue)
	GetListener()
	// just create queue
	http.HandleFunc("/", handleRequest)
	err := http.ListenAndServe(":7000", nil)
	if err != nil {
		panic(err)
	}

}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		addQueue(w, r)
		fmt.Println(queueList)
		return
	} else if r.Method == "GET" {
		getQueue(w, r)
		fmt.Println(queueList)
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
	q := GetQueue(queueName)
	if GetListener().Notify(queueName, queueValue) {
		return
	}

	q.Add(queueValue)
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
	queueValue := GetQueue(queueName).Pop()
	if queueValue == "" && timeout == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if queueValue != "" {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, queueValue)
		return
	}

	queueChannel := GetListener().Subscribe(queueName)
	for {
		select {
		case value := <-queueChannel:
			GetListener().Unsubscribe(queueName)
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, value)
			return
		case <-time.After(dTimeout * time.Second):
			GetListener().Unsubscribe(queueName)
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

}

func getQueueNameFromRequest(r *http.Request) string {
	return r.URL.Path[1:]
}

// queue implementation

type IQueue interface {
	Add(queueValue string)
	Pop() string
}

type Queue struct {
	list []string
	m    sync.Mutex
}

func (q *Queue) Add(queueValue string) {
	q.m.Lock()
	defer q.m.Unlock()

	q.list = append(q.list, queueValue)
}

func (q *Queue) Pop() string {
	q.m.Lock()
	defer q.m.Unlock()
	if len(q.list) == 0 {
		return ""
	}
	list := q.list
	q.list = q.list[1:]
	return list[0]
}

func GetQueue(queueName string) IQueue {
	v, err := queueList[queueName]
	if !err {
		v = &Queue{list: []string{}}
		queueList[queueName] = v
	}
	fmt.Println("test", v, err)
	return v
}

// listener implementation
type IListener interface {
	Subscribe(event string) chan string
	Unsubscribe(event string)
	Notify(event string, value string) bool
}

type listenChannel struct {
	listenerCount int64
	channel       chan string
}

type Listener struct {
	mu            sync.Mutex
	listenerCount map[string]listenChannel
}

func (l *Listener) Subscribe(event string) chan string {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ex := l.listenerCount[event]
	if !ex {
		v = listenChannel{
			listenerCount: 0,
			channel:       make(chan string),
		}
	}
	v.listenerCount++
	l.listenerCount[event] = v
	return v.channel
}

func (l *Listener) Unsubscribe(event string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ex := l.listenerCount[event]
	if !ex || v.listenerCount == 0 {
		panic(event + " event not exits, you should subscribe firstly")
	}

	v.listenerCount--
	l.listenerCount[event] = v
}

func (l *Listener) Notify(event string, value string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ex := l.listenerCount[event]
	if !ex || v.listenerCount == 0 {
		return false
	}
	v.channel <- value
	return true
}

func GetListener() IListener {
	if listener == nil {
		listener = &Listener{
			listenerCount: make(map[string]listenChannel),
		}
	}
	return listener
}
