package sendgrid

import (
	"github.com/sendgrid/rest"
)

// sendGridOptions for CreateRequest
type sendGridOptions struct {
	Key      string
	Endpoint string
	Host     string
	Subuser  string
}

// GetRequest
// @return [Request] a default request object
func GetRequest(key, endpoint, host string) rest.Request {
	return createSendGridRequest(sendGridOptions{key, endpoint, host, ""})
}

// GetRequestSubuser like GetRequest but with On-Behalf of Subuser
// @return [Request] a default request object
func GetRequestSubuser(key, endpoint, host, subuser string) rest.Request {
	return createSendGridRequest(sendGridOptions{key, endpoint, host, subuser})
}

// createSendGridRequest create Request
// @return [Request] a default request object
func createSendGridRequest(sgOptions sendGridOptions) rest.Request {
	options := options{
		"Bearer " + sgOptions.Key,
		sgOptions.Endpoint,
		sgOptions.Host,
		sgOptions.Subuser,
	}

	if options.Host == "" {
		options.Host = "https://api.sendgrid.com"
	}

	return requestNew(options)
}

// NewSendClient constructs a new Twilio SendGrid client given an API key
func NewSendClient(key string) *Client {
	request := GetRequest(key, "/v3/mail/send", "")
	request.Method = "POST"
	return &Client{request}
}
