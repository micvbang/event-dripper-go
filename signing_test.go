package eventdripper_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/micvbang/go-helpy/timey"
	"github.com/stretchr/testify/require"
	"gitlab.com/micvbang/event-dripper/pkg/eventdripper"
)

func TestWebhookSigning(t *testing.T) {
	expectedTime := time.Now()
	expectedSecret := []byte("im a secret")
	expectedPayload := []byte("im a payload")
	expectedSignature := eventdripper.ComputeSignature(expectedSecret, expectedTime, expectedPayload)

	tests := map[string]struct {
		correct   bool
		signature []byte
		secret    []byte
		payload   []byte
		t         time.Time
	}{
		"works": {
			correct:   true,
			payload:   expectedPayload,
			secret:    expectedSecret,
			signature: expectedSignature,
			t:         expectedTime,
		},
		"wrong secret": {
			correct:   false,
			payload:   expectedPayload,
			secret:    []byte("im wrong secret"),
			signature: expectedSignature,
			t:         expectedTime,
		},
		"wrong time": {
			correct:   false,
			payload:   expectedPayload,
			secret:    expectedSecret,
			signature: expectedSignature,
			t:         timey.AddHours(expectedTime, 5),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.correct {
				require.Equal(t, test.signature, eventdripper.ComputeSignature(test.secret, test.t, test.payload))
			} else {
				require.NotEqual(t, test.signature, eventdripper.ComputeSignature(test.secret, test.t, test.payload))
			}
		})
	}
}

func TestConstructNotification(t *testing.T) {
	now := time.Now()
	signatureExpired := timey.AddDays(now, -1)

	const expectedSecret = "dont-reveal-me-im-secret"
	expectedPayload := toJSON(t, &eventdripper.Notification{
		TriggerName: "trigger-name",
		EntityID:    "entity-id",
	})
	expectedSignature := eventdripper.ComputeSignature([]byte(expectedSecret), now, expectedPayload)
	expectedHeader := eventdripper.MakeHTTPHeader(now, expectedSignature)

	tests := map[string]struct {
		err     error
		header  string
		payload []byte
		secret  string
	}{
		"happy path": {
			err:     nil,
			payload: expectedPayload,
			secret:  expectedSecret,
			header:  expectedHeader,
		},
		"signature too old": {
			err:     eventdripper.ErrSignatureTooOld,
			payload: expectedPayload,
			secret:  expectedSecret,
			header:  eventdripper.MakeHTTPHeader(signatureExpired, expectedSignature),
		},
		"invalid payload, signed correctly": {
			err:     eventdripper.ErrInvalidPayload,
			payload: []byte("invalid payload"),
			secret:  expectedSecret,
			header:  eventdripper.MakeHTTPHeader(now, eventdripper.ComputeSignature([]byte(expectedSecret), now, []byte("invalid payload"))),
		},
		"invalid signature, wrong payload": {
			err:     eventdripper.ErrNoValidSignature,
			payload: []byte("invalid payload"),
			secret:  expectedSecret,
			header:  expectedHeader,
		},
		"invalid signature, wrong secret": {
			err:     eventdripper.ErrNoValidSignature,
			payload: expectedPayload,
			secret:  "invalid secret",
			header:  expectedHeader,
		},
		"no header": {
			err:     eventdripper.ErrNoSignature,
			payload: expectedPayload,
			secret:  expectedSecret,
			header:  "",
		},
		"invalid header, bad signature": {
			err:     eventdripper.ErrNoValidSignature,
			payload: expectedPayload,
			secret:  expectedSecret,
			header:  fmt.Sprintf("t=%d,c=not-valid", now.Unix()),
		},
		"invalid header, bad format": {
			err:     eventdripper.ErrInvalidHeader,
			payload: expectedPayload,
			secret:  expectedSecret,
			header:  fmt.Sprintf("t=%d,missing-equality", now.Unix()),
		},
		"invalid header, bad timestamp": {
			err:     eventdripper.ErrInvalidHeader,
			payload: expectedPayload,
			secret:  expectedSecret,
			header:  "t=invalid-timestamp",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := eventdripper.ConstructNotification(test.payload, test.header, test.secret)
			require.Equal(t, test.err, err)
		})
	}
}

func toJSON(t *testing.T, v interface{}) []byte {
	bs, err := json.Marshal(v)
	require.NoError(t, err)
	return bs
}
