package connect

import (
	"fmt"

	"github.com/coder/websocket"
)

type SocketError struct {
	SysCode    string
	Msg        string
	StatusCode websocket.StatusCode
}

func (se SocketError) Error() string {
	return fmt.Sprintf("[code: %s] %s", se.SysCode, se.Msg)
}
