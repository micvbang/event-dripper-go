package eventdripper_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	eventdripper "github.com/micvbang/event-dripper-go"
	"github.com/stretchr/testify/require"
)

func TestClientAddEventHappyPath(t *testing.T) {
	const (
		expectedAPIKey    = "im api key"
		expectedEventName = "im event name"
		expectedEntityID  = "im entity id"
	)
	expectedData := []byte("im data")

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert
		require.Equal(t, expectedAPIKey, r.Header.Get("Authorization"))

		defer r.Body.Close()
		buf, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)

		payload := eventdripper.AddEventInput{}
		err = json.Unmarshal(buf, &payload)
		require.NoError(t, err)

		require.Equal(t, expectedEntityID, payload.EntityID)
		require.Equal(t, expectedEventName, payload.EventName)
		require.Equal(t, expectedData, payload.Data)

		w.WriteHeader(http.StatusCreated) // Client expects http 201
	})
	s := httptest.NewServer(h)

	client := eventdripper.NewClientWithHost(s.Client(), expectedAPIKey, s.URL)

	// Test
	err := client.AddEvent(expectedEntityID, expectedEventName, expectedData)
	require.NoError(t, err)
}
