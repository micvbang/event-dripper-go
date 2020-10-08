package eventdripper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	client HTTPDo
	host   string
	apiKey string
}

func NewClient(client HTTPDo, apiKey string) *Client {
	return NewClientWithHost(client, apiKey, "https://api.production.event-dripper.haps.pw")
}

func NewClientWithHost(client HTTPDo, apiKey string, host string) *Client {
	return &Client{
		client: client,
		apiKey: apiKey,
		host:   host,
	}
}

type HTTPDo interface {
	Do(req *http.Request) (*http.Response, error)
}

type AddEventInput struct {
	EventName string `json:"event_name"`
	EntityID  string `json:"entity_id"`
	Data      []byte `json:"data"`
}

func (c *Client) AddEvent(entityID string, eventName string, data []byte) error {
	url := fmt.Sprintf("%s/api/event", c.host)

	buf := bytes.NewBuffer(nil)
	err := json.NewEncoder(buf).Encode(&AddEventInput{
		EventName: eventName,
		EntityID:  entityID,
		Data:      data,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", c.apiKey)

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	switch res.StatusCode {
	case http.StatusCreated:
		return nil

	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized")

	default:
		return fmt.Errorf("failed to create event; status code %d", res.StatusCode)
	}
}
