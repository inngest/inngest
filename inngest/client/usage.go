package client

import "time"

type Usage struct {
	Period string      `json:"period"`
	Range  string      `json:"range"`
	Total  int         `json:"total"`
	Data   []UsageSlot `json:"data"`
}

type UsageSlot struct {
	Slot  time.Time `json:"slot"`
	Count int64     `json:"count"`
}
