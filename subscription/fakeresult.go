package subscription

import (
	"errors"
)

type fakeRequest map[string]*fakeResult

type fakeResult struct {
	Response []Item
	Error    error
}

func fakeResponse(uri string) []Item {
	response := fakeRequest{
		"/book/feed1/success": &fakeResult{
			Response: []Item{
				{
					Title:   "Three Jobless Freaks",
					Channel: "Flipkart",
					GUID:    "Test GUID1",
				},
			},
			Error: nil,
		},
		"/book/feed2/success": &fakeResult{
			Response: []Item{
				{
					Title:   "Five Point Someone",
					Channel: "Amazon",
					GUID:    "Test GUID2",
				},
			},
			Error: nil,
		},
		"/book/feed3/success": &fakeResult{
			Response: []Item{
				{
					Title:   "The Golden Leaf",
					Channel: "Amazon",
					GUID:    "Test GUID3",
				},
			},
			Error: nil,
		},
		"/book/feed/error": &fakeResult{
			Response: []Item{},
			Error:    errors.New("Mock error"),
		},
	}
	return response[uri].Response
}
