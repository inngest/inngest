package v1

#Runtime: #RuntimeDocker | #RuntimeHTTP

#RuntimeDocker: {
	type:       "docker"
	dockerfile: string | *""
}

#RuntimeHTTP: {
	type: "http"
	url:  string
}
