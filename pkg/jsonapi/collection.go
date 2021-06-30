package jsonapi

import (
	"errors"
)

type Collection struct {
	API      *Connection
	Data     []Resource
	Next     string
	Previous string
}

/*
GetNext
Return the next page of the paginated collection as pointed to by the
`.links.next` field in the {json:api} response
*/
func (c *Collection) GetNext() (Collection, error) {
	var result Collection
	if c.Next == "" {
		return result, errors.New("no next page")
	}
	result, err := c.API.listFromPath(c.Next)
	if err != nil {
		return result, err
	}
	return result, err
}

/*
GetPrevious
Return the previous page of the paginated collection as pointed to by the
`.links.previous` field in the {json:api} response
*/
func (c *Collection) GetPrevious() (Collection, error) {
	var result Collection
	if c.Previous == "" {
		return result, errors.New("no previous page")
	}
	result, err := c.API.listFromPath(c.Previous)
	if err != nil {
		return result, err
	}
	return result, err
}
