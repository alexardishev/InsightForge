package cdc

import (
	"encoding/json"
)

type CDCEvent struct {
	Event string `json:"event"`
	ID    int    `json:"id"`
}

func Dispatch(eventBytes []byte, analyticsHandler func(CDCEvent)) {
	var evt CDCEvent
	if err := json.Unmarshal(eventBytes, &evt); err != nil {
		return // логгировать
	}

	switch evt.Event {
	case "created":
		analyticsHandler(evt)
		// другие типы событий
	}
}
