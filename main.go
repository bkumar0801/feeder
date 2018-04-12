package main

import (
	"fmt"
	"time"

	s "github.com/rssfeedreader/subscription"
)

func main() {
	// Subscribe to some feeds, and create a merged update stream.
	merged := s.Merge(
		s.Subscribe(s.Fetch("/book/feed1/success")),
		s.Subscribe(s.Fetch("/book/feed2/success")),
		s.Subscribe(s.Fetch("/book/feed3/success")))

	// Close the subscriptions after some time.
	time.AfterFunc(3*time.Second, func() {
		fmt.Println("closed:", merged.Close())
	})

	// Print the stream.
	for it := range merged.Updates() {
		fmt.Println(it.Channel, it.Title)
	}

	//panic("show me the stacks")
}
