package livestream

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// livekitVideoClaims mirrors the "video" grant object expected by LiveKit.
type livekitVideoClaims struct {
	Room        string `json:"room"`
	RoomJoin    bool   `json:"roomJoin"`
	CanPublish  bool   `json:"canPublish"`
	CanSubscribe bool  `json:"canSubscribe"`
	CanPublishData bool `json:"canPublishData"`
}

type livekitClaims struct {
	Video    livekitVideoClaims `json:"video"`
	Identity string             `json:"sub"`
	Name     string             `json:"name,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken produces a LiveKit access token for the given room and identity.
// isHost=true grants publish rights; false grants subscribe-only rights.
// Falls back to a simulated token string if LIVEKIT_API_KEY is not set.
func GenerateToken(roomName, identity, displayName string, isHost bool) (string, error) {
	apiKey := os.Getenv("LIVEKIT_API_KEY")
	apiSecret := os.Getenv("LIVEKIT_API_SECRET")

	if apiKey == "" || apiSecret == "" {
		return fmt.Sprintf("simulated_token_%s_%s", roomName, identity), nil
	}

	now := time.Now()
	claims := livekitClaims{
		Video: livekitVideoClaims{
			Room:           roomName,
			RoomJoin:       true,
			CanPublish:     isHost,
			CanSubscribe:   true,
			CanPublishData: true,
		},
		Identity: identity,
		Name:     displayName,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    apiKey,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(6 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(apiSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign livekit token: %w", err)
	}
	return signed, nil
}

// LiveKitURL returns the LiveKit server WebSocket URL from env.
func LiveKitURL() string {
	if u := os.Getenv("LIVEKIT_URL"); u != "" {
		return u
	}
	return "wss://your-livekit-instance.livekit.cloud"
}
