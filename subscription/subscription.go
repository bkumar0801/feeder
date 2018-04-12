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
Fetcher ...
*/
type Fetcher interface {
	Fetch() (items []Item, next time.Time, err error)
}

/*
Fetch ...
*/
func Fetch(domain string) Fetcher {
	return nil
}

/*
Subscription ...
*/
type Subscription interface {
	Updates() <-chan Item // stream of Items
	Close() error         // shuts down the stream
}

// sub implements the Subscription interface.
type sub struct {
	fetcher Fetcher   // fetches items
	updates chan Item // delivers items to the user
	err     error
	closing chan chan error
}

/*
Subscribe ...
*/
func Subscribe(fetcher Fetcher) Subscription {
	s := &sub{
		fetcher: fetcher,
		updates: make(chan Item), // for Updates
	}
	go s.loop()
	//return s
	return nil //TODO: remove this line
} // converts Fetches to a stream

/*
Merge ...
*/
func Merge(subs ...Subscription) Subscription {
	return nil //TODO: remove this line
} // merges several streams

// loop fetches items using s.fetcher and sends them
// on s.updates.  loop exits when s.Close is called.
func (s *sub) loop() {
	var pending []Item // appended by fetch; consumed by send
	var next time.Time // initially January 1, year 0
	var err error
	for {
		var fetchDelay time.Duration // initally 0 (no delay)
		var first Item
		var updates chan Item
		if now := time.Now(); next.After(now) {
			fetchDelay = next.Sub(now)
		}
		startFetch := time.After(fetchDelay)

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

func (s *sub) Updates() <-chan Item {
	return s.updates
}
func (s *sub) Close() error {
	errc := make(chan error)
	s.closing <- errc
	return <-errc
}
