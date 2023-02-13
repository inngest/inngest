{
	name: "api/user.created"
	data: {
		email: string
		plan:  "free" | "starter" | "pro"
	}
	user: {
		email:       string
		external_id: string
	}
	ts: int
}
