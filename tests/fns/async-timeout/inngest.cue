{
	name: "async timeout"
	id:   "async-timeout-fn-id"
	triggers: [{
		event: "api/user.created"
		definition: {
			format: "cue"
			synced: false
			def:    "file://./events/api-user-created.cue"
		}
	}]
	steps: {
		"step-1": {
			id:   "step-1"
			path: "file://./steps/step-1"
			name: "async timeout"
			runtime: {
				type: "docker"
			}
			after: [{
				step: "$trigger"
				async: {
					ttl:       "10s"
					event:     "api/account.connected"
					onTimeout: true
				}
			}]
		}
	}

}
