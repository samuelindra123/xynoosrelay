package messaging

import (
	"net/http/httptest"
	"testing"
)

func TestHistoryPage(t *testing.T) {
	for _, tc := range []struct {
		query   string
		before  int64
		limit   int
		invalid bool
	}{
		{"", 0, 100, false}, {"?before_sequence=101&limit=50", 101, 50, false},
		{"?before_sequence=0", 0, 0, true}, {"?before_sequence=-1", 0, 0, true},
		{"?before_sequence=9223372036854775808", 0, 0, true}, {"?before_sequence=abc", 0, 0, true},
		{"?limit=0", 0, 0, true}, {"?limit=101", 0, 0, true}, {"?limit=1.5", 0, 0, true},
	} {
		t.Run(tc.query, func(t *testing.T) {
			before, limit, err := historyPage(httptest.NewRequest("GET", "/messages"+tc.query, nil))
			if (err != nil) != tc.invalid {
				t.Fatalf("error = %v", err)
			}
			if !tc.invalid && (before != tc.before || limit != tc.limit) {
				t.Fatalf("got %d/%d", before, limit)
			}
		})
	}
}
