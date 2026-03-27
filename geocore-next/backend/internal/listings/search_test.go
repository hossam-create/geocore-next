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
			name:    "negative min_price",
			req:     listings.SearchRequest{MinPrice: fp(-10)},
			wantErr: true,
		},
		{
			name:    "negative max_price",
			req:     listings.SearchRequest{MaxPrice: fp(-5)},
			wantErr: true,
		},
		{
			name:    "min > max",
			req:     listings.SearchRequest{MinPrice: fp(500), MaxPrice: fp(100)},
			wantErr: true,
		},
		{
			name: "min equals max (valid)",
			req:  listings.SearchRequest{MinPrice: fp(100), MaxPrice: fp(100)},
		},
		{
			name: "only min_price set",
			req:  listings.SearchRequest{MinPrice: fp(50)},
		},
		{
			name: "only max_price set",
			req:  listings.SearchRequest{MaxPrice: fp(1000)},
		},
		{
			name:    "invalid geo: lat without lng",
			req:     listings.SearchRequest{Lat: fp(25.2), Radius: 10},
			wantErr: true,
		},
		{
			name:    "invalid geo: lng without lat",
			req:     listings.SearchRequest{Lng: fp(55.3), Radius: 10},
			wantErr: true,
		},
		{
			name: "valid geo search",
			req:  listings.SearchRequest{Lat: fp(25.2), Lng: fp(55.3), Radius: 50},
		},
		{
			name: "geo with lat+lng but no radius (uses default, Radius=0 means not explicitly set)",
			req:  listings.SearchRequest{Lat: fp(25.2), Lng: fp(55.3)},
		},
		{
			name:    "invalid condition",
			req:     listings.SearchRequest{Condition: "excellent"},
			wantErr: true,
		},
		{
			name: "valid condition: new",
			req:  listings.SearchRequest{Condition: "new"},
		},
		{
			name: "valid condition: like-new",
			req:  listings.SearchRequest{Condition: "like-new"},
		},
		{
			name: "valid condition: good",
			req:  listings.SearchRequest{Condition: "good"},
		},
		{
			name: "valid condition: fair",
			req:  listings.SearchRequest{Condition: "fair"},
		},
		{
			name: "valid condition: for-parts",
			req:  listings.SearchRequest{Condition: "for-parts"},
		},
		{
			name:    "invalid type",
			req:     listings.SearchRequest{Type: "swap"},
			wantErr: true,
		},
		{
			name: "valid type: sell",
			req:  listings.SearchRequest{Type: "sell"},
		},
		{
			name: "valid type: buy",
			req:  listings.SearchRequest{Type: "buy"},
		},
		{
			name: "valid type: rent",
			req:  listings.SearchRequest{Type: "rent"},
		},
		{
			name: "valid type: auction",
			req:  listings.SearchRequest{Type: "auction"},
		},
		{
			name: "valid type: service",
			req:  listings.SearchRequest{Type: "service"},
		},
		{
			name:    "invalid sort_by",
			req:     listings.SearchRequest{SortBy: "popularity"},
			wantErr: true,
		},
		{
			name: "valid sort_by: relevance",
			req:  listings.SearchRequest{SortBy: "relevance"},
		},
		{
			name: "valid sort_by: price_asc",
			req:  listings.SearchRequest{SortBy: "price_asc"},
		},
		{
			name: "valid sort_by: price_desc",
			req:  listings.SearchRequest{SortBy: "price_desc"},
		},
		{
			name: "valid sort_by: date",
			req:  listings.SearchRequest{SortBy: "date"},
		},
		{
			name: "valid sort_by: distance",
			req:  listings.SearchRequest{SortBy: "distance"},
		},
		{
			name: "empty sort_by (uses default)",
			req:  listings.SearchRequest{SortBy: ""},
		},
		{
			name: "all filters combined (valid)",
			req: listings.SearchRequest{
				Query:     "iPhone",
				Condition: "new",
				Type:      "sell",
				MinPrice:  fp(100),
				MaxPrice:  fp(2000),
				Lat:       fp(25.2),
				Lng:       fp(55.3),
				Radius:    25,
				City:      "Dubai",
				Country:   "UAE",
				SortBy:    "price_asc",
				Page:      2,
				PerPage:   10,
			},
		},
		{
			name: "zero price range (both zero, valid)",
			req:  listings.SearchRequest{MinPrice: fp(0), MaxPrice: fp(0)},
		},
		// Status filter validation — only publicly safe statuses are allowed
		{
			name:    "invalid status: deleted",
			req:     listings.SearchRequest{Status: "deleted"},
			wantErr: true,
		},
		{
			name:    "invalid status: published",
			req:     listings.SearchRequest{Status: "published"},
			wantErr: true,
		},
		{
			name:    "invalid status: draft (owner-internal, not publicly searchable)",
			req:     listings.SearchRequest{Status: "draft"},
			wantErr: true,
		},
		{
			name:    "invalid status: pending (owner-internal, not publicly searchable)",
			req:     listings.SearchRequest{Status: "pending"},
			wantErr: true,
		},
		{
			name: "valid status: active",
			req:  listings.SearchRequest{Status: "active"},
		},
		{
			name: "valid status: sold",
			req:  listings.SearchRequest{Status: "sold"},
		},
		{
			name: "valid status: reserved",
			req:  listings.SearchRequest{Status: "reserved"},
		},
		{
			name: "valid status: expired",
			req:  listings.SearchRequest{Status: "expired"},
		},
		{
			name: "empty status (defaults to active, no error)",
			req:  listings.SearchRequest{Status: ""},
		},
		// Price range with max=0 edge case
		{
			name:    "min_price=100 max_price=0 (min exceeds max)",
			req:     listings.SearchRequest{MinPrice: fp(100), MaxPrice: fp(0)},
			wantErr: true,
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

	// Same point — should be 0
	dist3 := listings.HaversineKM(25.0, 55.0, 25.0, 55.0)
	assert.InDelta(t, 0, dist3, 0.001)

	// Antipodal points — should be ~20015 km (half Earth circumference)
	dist4 := listings.HaversineKM(0, 0, 0, 180)
	assert.InDelta(t, 20015, dist4, 100)
}
