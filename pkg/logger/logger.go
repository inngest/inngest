package log

type Options struct {
	Pretty bool
}

type Message struct {
	Object  string
	Action  string
	Msg     string
	Context interface{}
}
