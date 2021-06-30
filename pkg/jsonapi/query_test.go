package jsonapi

import (
	"net/url"
	"strings"
	"testing"
)

func TestEncode(t *testing.T) {
	testCases := []struct {
		query    Query
		expected string
	}{
		{Query{Filters: map[string]string{"color": "red"}},
			"filter[color]=red"},
		{Query{Filters: map[string]string{"age__gt": "15"}},
			"filter[age][gt]=15"},
		{Query{Includes: []string{"aaa", "bbb"}},
			"include=aaa,bbb"},
		{Query{Extras: map[string]string{"limit": "15"}}, "limit=15"},
	}

	for _, testCase := range testCases {
		query := testCase.query
		expected := testCase.expected
		expected = url.QueryEscape(expected)
		expected = strings.ReplaceAll(expected, "%3D", "=")
		expected = strings.ReplaceAll(expected, "%26", "&")

		if query.Encode() != expected {
			t.Errorf("Query %s generated querystring '%s', expected '%s'",
				query, query.Encode(), expected)
		}
	}
}
