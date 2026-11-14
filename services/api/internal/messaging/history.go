package messaging

import (
	"net/http"
	"strconv"

	"github.com/xynoss/relay/services/api/internal/response"
)

// Sequence cursors are conversation scoped; authorization remains mandatory.
func historyPage(r *http.Request) (before int64, limit int, err error) {
	limit = 100
	if value := r.URL.Query().Get("before_sequence"); value != "" {
		before, err = strconv.ParseInt(value, 10, 64)
		if err != nil || before <= 0 {
			return 0, 0, response.NewBadRequestError("before_sequence must be a positive integer")
		}
	}
	if value := r.URL.Query().Get("limit"); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 100 {
			return 0, 0, response.NewBadRequestError("limit must be between 1 and 100")
		}
	}
	return before, limit, nil
}
