package ss12000v2import

import (
	"context"
	"net/http"

	"github.com/Sambruk/windermere/ss12000v2"
)

func NewAPIKeyClient(url, clientSecret, apiKeyHeader string) (ss12000v2.ClientInterface, error) {
	apiKeyAdder := func(ctx context.Context, req *http.Request) error {
		req.Header.Set(apiKeyHeader, clientSecret)
		return nil
	}
	return ss12000v2.NewClient(url, ss12000v2.WithRequestEditorFn(apiKeyAdder))
}
