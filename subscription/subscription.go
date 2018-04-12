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
Subscription ...
*/
type Subscription interface {
	Updates() <-chan Item // stream of Items
	Close() error         // shuts down the stream
}

/*
Fetcher ...
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
	var pending []Item // appended by fetch; consumed by send
	var next time.Time // initially January 1, year 0
	var err error
	for {
		var fetchDelay time.Duration // initally 0 (no delay)
		if now := time.Now(); next.After(now) {
			fetchDelay = next.Sub(now)
		}
		startFetch := time.After(fetchDelay)

		var first Item
		var updates chan Item
		if len(pending) > 0 {
			first = pending[0]
			updates = s.updates // enable send case
		}

		select {
		case errc := <-s.closing:
			errc <- err
			close(s.updates) // tells receiver we're done
			return
		case <-startFetch:
			var fetched []Item
			fetched, next, err = s.fetcher.Fetch()
			if err != nil {
				next = time.Now().Add(10 * time.Second)
				break
			}
			pending = append(pending, fetched...)

		case updates <- first:
			pending = pending[1:]
		}
	}
}
