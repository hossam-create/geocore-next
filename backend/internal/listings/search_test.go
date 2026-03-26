package listings_test

import (
	"testing"

	"github.com/geocore-next/backend/internal/listings"
	"github.com/stretchr/testify/assert"
)

func fp(v float64) *float64 { return &v }

func TestSearchRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     listings.SearchRequest
		wantErr bool
	}{
		{
			name: "valid basic search",
			req:  listings.SearchRequest{Query: "iPhone", Page: 1, PerPage: 20},
		},
		{
			name:    "negative price",
			req:     listings.SearchRequest{MinPrice: fp(-10)},
			wantErr: true,
		},
		{
			name:    "min > max",
			req:     listings.SearchRequest{MinPrice: fp(500), MaxPrice: fp(100)},
			wantErr: true,
		},
		{
			name:    "invalid geo: lat without lng",
			req:     listings.SearchRequest{Lat: fp(25.2), Radius: 10},
			wantErr: true,
		},
		{
			name: "valid geo search",
			req:  listings.SearchRequest{Lat: fp(25.2), Lng: fp(55.3), Radius: 50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHaversineDistance(t *testing.T) {
	// Dubai Mall to Burj Khalifa — ~0.5 km
	dist := listings.HaversineKM(25.1972, 55.2796, 25.1975, 55.2742)
	assert.InDelta(t, 0.5, dist, 0.3)

	// Dubai to London — ~5500 km
	dist2 := listings.HaversineKM(25.2, 55.3, 51.5, -0.1)
	assert.InDelta(t, 5500, dist2, 300)
}
