# Inngest cli

A command line tool for working with [Inngest.com](https://www.inngest.com).

### Installation

```go install github.com/inngest/inngestctl```

### Usage

To do anything of value you must log in to inngest:

```
$ inngestctl login -u "your@email.example.com"
```

From there, you can run `inngestctl -h` to view help and usage information.

### Settings


```bash
inngest login -u -p # optional --ttl expiration

inngest workspace
	- list
	- create

inngest event
	- list
	- show [event] # show only workflows / type of event / etc.
	- usage [event] # show usage
	- realtime [event] # show realtime dashboard

inngest workflow
	- list
	- create # create a new empty workflow
	- show [workflow name] # show information about a current workflow
	- history [wofkflow name] # show previous runs
	- logs [workflow run]
	- diff [workflow.cue]
	- deploy [workflow.cue]
	- validate [workflow.cue]

inngest action
	- create # create a new empty action
	- diff [action.cue]
	- deploy [action.cue]
	- validate [action.cue]

inngest debug -workflow -event
```
