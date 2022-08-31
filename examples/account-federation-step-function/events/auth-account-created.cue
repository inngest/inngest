{
	name: "auth/account.created"
	data: {
		// Data contains all of the information from the associated event.  This is an
		// example event that is created when a user signs up.
		account_id:  string
		method:      string
		plan_name:   string
		subscribed?: bool
	}
	user: {
		// This object contains information for user attribution, allowing us to assign
		// the event and associated function runs to this specific user.  This allows us
		// to do fun things like user-specific debugging, or create K/V stores per user
		// for use in future functions.
		email:       string
		external_id: string
		plan_name:   string
	}
	v:  "1"
	ts: int
}
