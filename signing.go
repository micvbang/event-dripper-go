package eventdripper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ComputeSignature computes the signature of a webhook, which through a
// shared secret allows users to verify that the sender knows the secret.
func ComputeSignature(secret []byte, t time.Time, payload []byte) []byte {
	mac := hmac.New(sha256.New, secret)

	mac.Write([]byte(fmt.Sprintf("%d", t.Unix())))
	mac.Write([]byte("."))
	mac.Write(payload)

	return mac.Sum(nil)
}

// ConstructNotification constructs a Notification, verifying the payload using
// the given header and secret.
func ConstructNotification(payload []byte, header string, secret string) (Notification, error) {
	signedHeader, err := parseSignatureHeader(header)
	if err != nil {
		return Notification{}, err
	}

	if time.Since(signedHeader.timestamp) > MaxSignatureAge {
		return Notification{}, ErrSignatureTooOld
	}

	expectedSignature := ComputeSignature([]byte(secret), signedHeader.timestamp, payload)
	for _, gotSignature := range signedHeader.signatures {
		if hmac.Equal(expectedSignature, gotSignature) {
			notification := Notification{}
			err := json.Unmarshal(payload, &notification)
			if err != nil {
				return Notification{}, ErrInvalidPayload
			}

			return notification, nil // Valid signature and payload
		}
	}

	return Notification{}, ErrNoValidSignature
}

func MakeHTTPHeader(t time.Time, signature []byte) string {
	return fmt.Sprintf("t=%d,%s=%s", t.Unix(), SigningVersion, hex.EncodeToString(signature))
}

const (
	// HeaderKey is the HTTP header key used when sending the signature.
	HeaderKey = "EventDripper-Signature"

	// MaxSignatureAge indicates that signatures older than this will be rejected by ConstructEvent.
	MaxSignatureAge time.Duration = 300 * time.Second

	// SigningVersion represents the version of the signature currently used.
	SigningVersion string = "v1"
)

type signedHeader struct {
	timestamp  time.Time
	signatures [][]byte
}

var (
	ErrInvalidHeader    = errors.New("webhook has invalid EventDripper-Signature header")
	ErrNoValidSignature = errors.New("webhook had no valid signature")
	ErrNoSignature      = errors.New("webhook has no EventDripper-Signature header")
	ErrInvalidPayload   = errors.New("webhook has invalid payload")
	ErrSignatureTooOld  = errors.New("signature timestamp wasn't within tolerance")
)

func parseSignatureHeader(header string) (signedHeader, error) {
	if header == "" {
		return signedHeader{}, ErrNoSignature
	}

	sh := signedHeader{}

	// Signed header looks like "t=1601036356,v1=FOO,v1=BAR,v0=BAZ"
	pairs := strings.Split(header, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) != 2 {
			return sh, ErrInvalidHeader
		}
		name, value := parts[0], parts[1]

		switch name {
		case "t":
			timestamp, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return sh, ErrInvalidHeader
			}
			sh.timestamp = time.Unix(timestamp, 0)

		case SigningVersion:
			sig, err := hex.DecodeString(value)
			if err != nil {
				continue // Skip invalid signatures
			}

			sh.signatures = append(sh.signatures, sig)

		default:
			continue // Ignore unknown parts of the header
		}
	}

	if len(sh.signatures) == 0 {
		return sh, ErrNoValidSignature
	}

	return sh, nil
}
