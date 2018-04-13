package subscription

import (
	"time"
)

/*
Item ...
*/
type Item struct {
	Title, Channel, GUID string // a subset of RSS fields
}

/*
Subscription ...creates streams of items
shuts down the stream
*/
type Subscription interface {
	Updates() <-chan Item // stream of Items
	Close() error         // shuts down the stream
}

/*
Fetcher ...fetches items from web
*/
type Fetcher interface {
	Fetch() (items []Item, next time.Time, err error)
}

// sub implements the Subscription interface.
type sub struct {
	fetcher Fetcher   // fetches items
	updates chan Item // delivers items to the user
	closing chan chan error
}

func (s *sub) Updates() <-chan Item {
	return s.updates
}

/*
Close ...asks loop to exit and waits for a response.
*/
func (s *sub) Close() error {
	errc := make(chan error)
	s.closing <- errc
	return <-errc
}

/*
Fetch ...
*/
func Fetch(domain string) Fetcher {
	return fakeFetch(domain)
}

/*
Subscribe ...
*/
func Subscribe(fetcher Fetcher) Subscription {
	s := &sub{
		fetcher: fetcher,
		updates: make(chan Item), // for Updates
		closing: make(chan chan error),
	}
	go s.loop()
	return s
} // converts Fetches to a stream

type merge struct {
	subs    []Subscription
	updates chan Item
	quit    chan struct{}
	errs    chan error
}

func (m *merge) Updates() <-chan Item {
	return m.updates
}

func (m *merge) Close() (err error) {
	close(m.quit)
	for range m.subs {
		if e := <-m.errs; e != nil {
			err = e
		}
	}
	close(m.updates)
	return
}

/*
Merge ...
*/
func Merge(subs ...Subscription) Subscription {
	m := &merge{
		subs:    subs,
		updates: make(chan Item),
		quit:    make(chan struct{}),
		errs:    make(chan error),
	}
	for _, sub := range subs {
		go func(s Subscription) {
			for {
				var it Item
				select {
				case it = <-s.Updates():
				case <-m.quit:
					m.errs <- s.Close()
					return
				}
				select {
				case m.updates <- it:
				case <-m.quit:
					m.errs <- s.Close()
					return
				}
			}
		}(sub)
	}
	return m
}

// loop fetches items using s.fetcher and sends them
// on s.updates.  loop exits when s.Close is called.
func (s *sub) loop() {
	const maxPending = 10 // less than 10% pending items are allowed
	type fetchResult struct {
		fetched []Item
		next    time.Time
		err     error
	}
	var fetchDone chan fetchResult // if non-nil, Fetch is running
	var pending []Item             // appended by fetch; consumed by send
	var next time.Time             // initially January 1, year 0
	var err error
	var seen = make(map[string]bool) // set of item.GUIDs to remove duplicates
	for {
		var fetchDelay time.Duration // initally 0 (no delay)
		if now := time.Now(); next.After(now) {
			fetchDelay = next.Sub(now)
		}
		var startFetch <-chan time.Time
		if fetchDone == nil && len(pending) < maxPending { //fetch asynchronously when there is less items pending
			startFetch = time.After(fetchDelay) // enable fetch case
		}

		var first Item
		var updates chan Item
		if len(pending) > 0 {
			first = pending[0]
			updates = s.updates // enable send case
		}

		select {
		case errc := <-s.closing: //reply with the Fetch error and exit
			errc <- err
			close(s.updates) // tells receiver we're done
			return
		case <-startFetch: // start fetch when there are less tha 10% items in pending and when fetch completed
			fetchDone = make(chan fetchResult, 1)  //channel to make fetch asynchronous
			go func() {
				fetched, next, err := s.fetcher.Fetch()
				fetchDone <- fetchResult{fetched, next, err}
			}()
		case result := <-fetchDone:
			fetchDone = nil
			// Use result.fetched, result.next, result.err
			fetched := result.fetched
			next, err = result.next, result.err
			if err != nil {
				next = time.Now().Add(10 * time.Second)
				break
			}
			for _, item := range fetched {
				if id := item.GUID; !seen[id] {
					pending = append(pending, item)
					seen[id] = true
				}
			}

		case updates <- first:
			pending = pending[1:]
		}
	}
}
