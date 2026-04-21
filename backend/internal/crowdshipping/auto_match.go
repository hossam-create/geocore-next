package crowdshipping

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/pkg/reputation"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TravelerMatch struct {
	TravelerID   uuid.UUID `json:"traveler_id"`
	TripID       uuid.UUID `json:"trip_id"`
	Score        float64   `json:"score"`
	CountryMatch float64   `json:"country_match"`
	TimeMatch    float64   `json:"time_match"`
	Reputation   float64   `json:"reputation"`
	PriceFit     float64   `json:"price_fit"`
	CanDeliver   bool      `json:"can_deliver"`
}

func FindBestTravelersForRequest(db *gorm.DB, requestID uuid.UUID, limit int) ([]TravelerMatch, error) {
	if limit <= 0 {
		limit = 10
	}
	var dr DeliveryRequest
	if err := db.Where("id=?", requestID).First(&dr).Error; err != nil {
		return nil, fmt.Errorf("request not found: %w", err)
	}
	var trips []Trip
	db.Where("origin_country=? AND dest_country=? AND status=? AND departure_date>?",
		dr.PickupCountry, dr.DeliveryCountry, TripStatusActive, time.Now()).Find(&trips)
	var exclIDs []uuid.UUID
	db.Model(&TravelerOffer{}).Where("delivery_request_id=?", requestID).
		Distinct("traveler_id").Pluck("traveler_id", &exclIDs)
	excl := map[uuid.UUID]bool{}
	for _, id := range exclIDs {
		excl[id] = true
	}
	matches := make([]TravelerMatch, 0, len(trips))
	for _, t := range trips {
		if excl[t.TravelerID] {
			continue
		}
		cm := countryScore(t, dr)
		tm := timeScore(t, dr)
		rep := repScore(db, t.TravelerID)
		pf := priceScore(t, dr)
		sc := cm*0.40 + tm*0.25 + (rep/100)*0.20 + pf*0.15

		// Apply trust multiplier: higher trust = better ranking
		trust := GetTrustScore(db, t.TravelerID)
		sc = ApplyTrustScoreToMatching(sc, trust)
		if strings.EqualFold(t.OriginCity, dr.PickupCity) && strings.EqualFold(t.DestCity, dr.DeliveryCity) {
			sc += 0.05
		}
		can := dr.ItemWeight == nil || t.AvailableWeight >= *dr.ItemWeight
		matches = append(matches, TravelerMatch{t.TravelerID, t.ID, sc, cm, tm, rep, pf, can})
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Score > matches[j].Score })
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func countryScore(t Trip, dr DeliveryRequest) float64 {
	if eq(t.OriginCountry, dr.PickupCountry) && eq(t.DestCountry, dr.DeliveryCountry) {
		return 1.0
	}
	if eq(t.OriginCountry, dr.PickupCountry) || eq(t.DestCountry, dr.DeliveryCountry) {
		return 0.5
	}
	return 0
}

func timeScore(t Trip, dr DeliveryRequest) float64 {
	if dr.Deadline == nil {
		return 1.0
	}
	w := time.Until(*dr.Deadline).Hours()
	if w <= 0 {
		return 0
	}
	return 1.0 - clamp(time.Until(t.DepartureDate).Hours()/w, 0, 1)
}

func repScore(db *gorm.DB, id uuid.UUID) float64 {
	if rp, err := reputation.Get(db, id.String()); err == nil {
		return rp.Score
	}
	return 50.0
}

func priceScore(t Trip, dr DeliveryRequest) float64 {
	if dr.Reward <= 0 || dr.ItemWeight == nil || *dr.ItemWeight <= 0 {
		return 0.5
	}
	est := t.PricePerKg**dr.ItemWeight + t.BasePrice
	if est <= 0 {
		return 0.5
	}
	r := est / dr.Reward
	if r <= 1.0 {
		return 1.0
	}
	if r <= 1.5 {
		return 1.0 - (r - 1.0)
	}
	return 0
}

func eq(a, b string) bool { return strings.EqualFold(a, b) }
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// AutoNotifyTopTravelers sends notifications to top 5 matched travelers.
func AutoNotifyTopTravelers(db *gorm.DB, notifSvc *notifications.Service, requestID uuid.UUID) {
	matches, err := FindBestTravelersForRequest(db, requestID, 5)
	if err != nil {
		slog.Error("auto_match: find failed", "error", err)
		return
	}
	for _, m := range matches {
		if notifSvc != nil {
			go notifSvc.Notify(notifications.NotifyInput{
				UserID: m.TravelerID, Type: "new_delivery_match",
				Title: "New Delivery Request Match!",
				Body:  "A delivery request matches your trip — submit an offer",
				Data:  map[string]string{"request_id": requestID.String(), "score": fmt.Sprintf("%.2f", m.Score)},
			})
		}
	}
	slog.Info("auto_match: notified travelers", "request_id", requestID, "count", len(matches))
}
