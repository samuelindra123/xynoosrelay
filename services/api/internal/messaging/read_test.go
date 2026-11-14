package messaging

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeReadRejectsInvalidPayloads(t *testing.T) {
	for _, body := range []string{`{}`, `{"last_read_sequence":null}`, `{"last_read_sequence":-1}`, `{"last_read_sequence":1.5}`, `{"last_read_sequence":9223372036854775808}`, `{"last_read_sequence":1,"actor_id":"other"}`, `{} {}`, `null`, strings.Repeat("x", 2048)} {
		t.Run(body[:min(len(body), 50)], func(t *testing.T) {
			_, err := decodeRead(httptest.NewRequest("POST", "/read", strings.NewReader(body)), httptest.NewRecorder())
			if err == nil {
				t.Fatal("invalid payload accepted")
			}
		})
	}
}

func TestDecodeReadAcceptsSequence(t *testing.T) {
	state, err := decodeRead(httptest.NewRequest("POST", "/read", strings.NewReader(`{"last_read_sequence":25}`)), httptest.NewRecorder())
	if err != nil || state.LastReadSequence != 25 {
		t.Fatalf("got %+v, %v", state, err)
	}
}
