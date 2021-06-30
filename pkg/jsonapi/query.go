package jsonapi

import (
	"fmt"
	"net/url"
	"strings"
)

type Query struct {
	Filters  map[string]string
	Includes []string
	Extras   map[string]string
}

/*
Encode
Converts a Query object to a string that's ready to be used as GET variables
for {json:api} requests.
*/
func (q Query) Encode() string {
	result := make(url.Values)
	if q.Filters != nil {
		for key, value := range q.Filters {
			finalKey := "filter"
			for _, part := range strings.Split(key, "__") {
				finalKey = finalKey + fmt.Sprintf("[%s]", part)
			}
			result.Add(finalKey, value)
		}
	}
	if q.Includes != nil {
		result.Add("include", strings.Join(q.Includes, ","))
	}
	if q.Extras != nil {
		for key, value := range q.Extras {
			result.Add(key, value)
		}
	}
	return result.Encode()
}
