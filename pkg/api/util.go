package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type apiResponse struct {
	StatusCode int    `json:"status"`
	Message    string `json:"message"`
	Error      string `json:"error,omitempty"`
}

func (a API) writeResponse(w http.ResponseWriter, h apiResponse) {
	w.WriteHeader(h.StatusCode)

	byt, err := json.Marshal(h)
	if err != nil {
		fmt.Println("Error marshalling response:", err)
	}

	_, err = w.Write(byt)
	if err != nil {
		fmt.Println("Error writing response:", err)
	}
	_, _ = w.Write([]byte("\n"))
}
