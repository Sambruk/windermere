package ss12000v2import

import (
	"fmt"

	"github.com/Sambruk/windermere/ss12000v2"
)

func NewClient(conf APIConfiguration) (ss12000v2.ClientInterface, error) {
	switch conf.Authentication {
	case AuthEduCloud:
		return NewEduCloudClient(conf.URL, conf.ClientId, conf.ClientSecret)
	case AuthAPIKey:
		return NewAPIKeyClient(conf.URL, conf.ClientSecret, conf.APIKeyHeader)
	}
	return nil, fmt.Errorf("unrecognized authentication type: %s", string(conf.Authentication))
}
