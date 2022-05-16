package logger

type Logger interface {
	Log(msg Message)
}

type Options struct {
	Pretty bool
}

type Message struct {
	Object  string
	Action  string
	Msg     string
	Context interface{}
}
