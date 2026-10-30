package server

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// feed polls one collector and fans its readings out to subscribers. A new
// subscriber gets the last size readings straight away, so a fresh page
// draws a full chart (dashdot's ReplaySubject). Unless it runs in the
// background, a feed only polls while someone is subscribed.
type feed[T any] struct {
	name       string
	enabled    bool
	background bool
	size       int
	every      time.Duration
	collect    func() (T, error)

	mu     sync.Mutex
	buf    []T
	subs   map[int]func(T)
	nextID int
	stop   chan struct{}
}

func newFeed[T any](name string, enabled, background bool, size, everyMs float64, collect func() (T, error)) *feed[T] {
	f := &feed[T]{
		name: name, enabled: enabled, background: background,
		size: max(1, int(size)), every: time.Duration(max(100, everyMs)) * time.Millisecond,
		collect: collect, subs: map[int]func(T){},
	}
	if enabled && background {
		f.mu.Lock()
		f.start()
		f.mu.Unlock()
	}
	return f
}

// start runs with f.mu held.
func (f *feed[T]) start() {
	if f.stop != nil {
		return
	}
	stop := make(chan struct{})
	f.stop = stop
	go f.run(stop)
}

func (f *feed[T]) run(stop chan struct{}) {
	first := true
	t := time.NewTicker(f.every)
	defer t.Stop()
	for {
		v, err := f.collect()
		if err != nil {
			log.Printf("%s: %v", f.name, err)
		} else {
			if first {
				b, _ := json.Marshal(v)
				log.Printf("First measurement [%s]: %s", f.name, b)
				first = false
			}
			f.publish(stop, v)
		}
		select {
		case <-stop:
			return
		case <-t.C:
		}
	}
}

func (f *feed[T]) publish(stop chan struct{}, v T) {
	f.mu.Lock()
	if f.stop != stop {
		f.mu.Unlock()
		return
	}
	f.buf = append(f.buf, v)
	if len(f.buf) > f.size {
		f.buf = f.buf[len(f.buf)-f.size:]
	}
	subs := make([]func(T), 0, len(f.subs))
	for _, s := range f.subs {
		subs = append(subs, s)
	}
	f.mu.Unlock()
	for _, s := range subs {
		s(v)
	}
}

// subscribe replays the buffer to fn, then sends it each new reading until
// the returned func is called.
func (f *feed[T]) subscribe(fn func(T)) func() {
	if !f.enabled {
		return func() {}
	}
	f.mu.Lock()
	f.start()
	// Under the lock, so a reading published meanwhile can't overtake it.
	for _, v := range f.buf {
		fn(v)
	}
	id := f.nextID
	f.nextID++
	f.subs[id] = fn
	f.mu.Unlock()
	return func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		delete(f.subs, id)
		if !f.background && len(f.subs) == 0 && f.stop != nil {
			close(f.stop)
			f.stop = nil
			f.buf = nil
		}
	}
}

// current is the latest reading, or a fresh one when the feed isn't polling.
func (f *feed[T]) current() (T, error) {
	f.mu.Lock()
	if n := len(f.buf); n > 0 {
		v := f.buf[n-1]
		f.mu.Unlock()
		return v, nil
	}
	f.mu.Unlock()
	return f.collect()
}
