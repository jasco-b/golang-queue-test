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

var q IQueue
var listener IListener

func main() {
	// just create queue
	GetQueue()
	http.HandleFunc("/", handleRequest)
	err := http.ListenAndServe(":7000", nil)
	if err != nil {
		panic(err)
	}

}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		addQueue(w, r)
		fmt.Println(GetQueue().List())
		return
	} else if r.Method == "GET" {
		getQueue(w, r)
		fmt.Println(GetQueue().List())
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
	q := GetQueue()
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
	queueValue := GetQueue().Get(queueName)
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
	if len(r.URL.Path) > 0 && r.URL.Path[:1] == "/" {
		return r.URL.Path[1:]
	}
	return r.URL.Path
}

// queue implementation

type IQueue interface {
	Add(queueName string, queueValue string)
	Get(queueName string) string
	List() map[string][]string
}

type Queue struct {
	list map[string][]string
	m    sync.Mutex
}

func (q *Queue) Add(queueName string, queueValue string) {
	q.m.Lock()
	defer q.m.Unlock()
	var v []string
	v, ex := q.list[queueName]
	if !ex {
		v = []string{}
	}
	// breaks solid (
	if GetListener().Notify(queueName, queueValue) {
		return
	}
	v = append(v, queueValue)
	q.list[queueName] = v
}

func (q *Queue) Get(queueName string) string {
	q.m.Lock()
	defer q.m.Unlock()
	v, ex := q.list[queueName]
	if !ex {
		return ""
	}
	itemCount := len(v)
	if itemCount == 0 {
		return ""
	}
	l := v[1:]
	q.list[queueName] = l
	return v[0]
}

func (q *Queue) List() map[string][]string {
	return q.list
}

func GetQueue() IQueue {
	if q == nil {
		q = &Queue{list: make(map[string][]string)}
	}
	GetListener()
	return q
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
