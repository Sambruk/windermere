package ss12000v2import

type AuthenticationType string

const (
	AuthEduCloud AuthenticationType = "EduCloud"
	AuthAPIKey   AuthenticationType = "APIKey"
)

// The APIConfiguration defines how to connect and authenticate
// to a specific SS12000 v2.1 API. From this we can create a
// SS12000 Client which knows how to connect and authenticate.
type APIConfiguration struct {
	URL            string             // The base URL of the SS12000 API
	Authentication AuthenticationType // Method used to authenticate
	ClientId       string             // Used for EduCloud authentication
	ClientSecret   string             // Used for EduCloud and APIKey authentication
	APIKeyHeader   string             // Used for APIKey authentication
}
