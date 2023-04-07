package lib

import (
	"sync"
)

var q IQueue

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
