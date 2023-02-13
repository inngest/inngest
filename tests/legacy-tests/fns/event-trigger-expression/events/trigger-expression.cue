{
	name: "test/trigger-expression"
	data: {
		ok: bool
		cart_items?: [...{
			price: int
		}]
	}
	user: {
		email?: string
	}
	v?:  string
	ts?: number
}
