package lib

import "sync"

var listener IListener

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
