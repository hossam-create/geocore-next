package listings

import (
        "fmt"
        "math"
)

// HaversineKM returns the great-circle distance in kilometres between two
// geographic coordinates.
func HaversineKM(lat1, lon1, lat2, lon2 float64) float64 {
        const earthR = 6371.0
        dLat := (lat2 - lat1) * math.Pi / 180
        dLon := (lon2 - lon1) * math.Pi / 180
        a := math.Sin(dLat/2)*math.Sin(dLat/2) +
                math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
                        math.Sin(dLon/2)*math.Sin(dLon/2)
        return earthR * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

var validSearchConditions = map[string]bool{
        "new": true, "like-new": true, "good": true, "fair": true, "for-parts": true,
}

var validSearchTypes = map[string]bool{
        "sell": true, "buy": true, "rent": true, "auction": true, "service": true,
}

// validSearchStatuses restricts public search to listing states that are
// safe to expose to all callers. Owner-internal states (draft, pending) are
// intentionally excluded so unauthenticated clients cannot enumerate them.
var validSearchStatuses = map[string]bool{
        "active": true, "sold": true, "reserved": true, "expired": true,
}

var validSortBys = map[string]bool{
        "relevance": true, "price_asc": true, "price_desc": true, "date": true, "distance": true,
}

func (r *SearchRequest) Validate() error {
        if r.MinPrice != nil && *r.MinPrice < 0 {
                return fmt.Errorf("price cannot be negative")
        }
        if r.MaxPrice != nil && *r.MaxPrice < 0 {
                return fmt.Errorf("price cannot be negative")
        }
        if r.MinPrice != nil && r.MaxPrice != nil && *r.MinPrice > *r.MaxPrice {
                return fmt.Errorf("min_price must be <= max_price")
        }
        if r.Radius > 0 && (r.Lat == nil || r.Lng == nil) {
                return fmt.Errorf("lat and lng required when radius is set")
        }
        if r.Condition != "" && !validSearchConditions[r.Condition] {
                return fmt.Errorf("invalid condition: must be one of new, like-new, good, fair, for-parts")
        }
        if r.Type != "" && !validSearchTypes[r.Type] {
                return fmt.Errorf("invalid type: must be one of sell, buy, rent, auction, service")
        }
        if r.Status != "" && !validSearchStatuses[r.Status] {
                return fmt.Errorf("invalid status: must be one of active, sold, reserved, expired")
        }
        if r.SortBy != "" && !validSortBys[r.SortBy] {
                return fmt.Errorf("invalid sort_by: must be one of relevance, price_asc, price_desc, date, distance")
        }
        if r.Page < 0 {
                return fmt.Errorf("page must be >= 0")
        }
        if r.PerPage < 0 {
                return fmt.Errorf("per_page must be >= 0")
        }
        return nil
}
