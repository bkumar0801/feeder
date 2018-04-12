package main

import (
	"fmt"
	"time"

	s "github.com/rssfeedreader/subscription"
)

func main() {
	// Subscribe to some feeds, and create a merged update stream.
	merged := s.Merge(
		s.Subscribe(s.Fetch("blog.fake.org")),
		s.Subscribe(s.Fetch("fakeblog.blogspot.com")),
		s.Subscribe(s.Fetch("developers.blogspot.com")))

	// Close the subscriptions after some time.
	time.AfterFunc(3*time.Second, func() {
		fmt.Println("closed:", merged.Close())
	})

	// Print the stream.
	for it := range merged.Updates() {
		fmt.Println(it.Channel, it.Title)
	}

	panic("show me the stacks")
}
