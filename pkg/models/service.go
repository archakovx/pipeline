package models

import "time"

//easyjson:skip
type Job func(in, out chan interface{})

type ExecuteRequest struct {
}

type ExecuteResponse struct {
	Data      struct{} `json:"data"`
	Error     bool     `json:"error"`
	ErrorText string   `json:"errorText"`
}

type Record struct {
	Ticker    string
	Price     float64
	Timestamp time.Time
}

type CandleInfo struct {
	Scope         string
	Ticker        string
	Date          time.Time
	StartingPoint float64
	EndingPoint   float64
	MinPrice      float64
	MaxPrice      float64
	Idx           int
}

type Key struct {
	Ticker string
	Date   time.Time
}
