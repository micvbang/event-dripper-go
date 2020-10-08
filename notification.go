package eventdripper

import "time"

type Notification struct {
	TriggerName string  `json:"trigger_name"`
	EntityID    string  `json:"entity_id"`
	Events      []Event `json:"events"`
}

type Event struct {
	At   time.Time `json:"at"`
	Name string    `json:"name"`
	Data []byte    `json:"data"`
}
