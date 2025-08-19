package inngest

type FunctionDriver struct {
	// URI represents how this function is invoked, eg https://example.com/api/inngest?step=foo.
	URI string `json:"uri"`

	// Metadata is additional data for the driver.  For example, the `http(s)` scheme can be used
	// for sync and async functions;  sync functions re-enter with a different driver format.
	//
	// This allows custom driver-specific data to be stored.
	Metadata map[string]any `json:"metadata"`
}

func Driver(f Function) string {
	url := f.URI()

	switch url.Scheme {
	case "http", "https":
		// HTTP can be one of async or sync.
		//
		// The Sync http driver is used to re-enter API-based sync functions.  This allows us to re-enter
		// ANY api, eg. "GET" or "PUT" API requests.
		return "http"
	case "ws", "wss":
		return "connect"
	default:
		return ""
	}
}
