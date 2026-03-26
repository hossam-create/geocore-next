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

func (r *SearchRequest) Validate() error {
        if r.MinPrice != nil && *r.MinPrice < 0 {
                return fmt.Errorf("price cannot be negative")
        }
        if r.MaxPrice != nil && *r.MaxPrice < 0 {
                return fmt.Errorf("price cannot be negative")
        }
        if r.MinPrice != nil && r.MaxPrice != nil && *r.MaxPrice > 0 && *r.MinPrice > *r.MaxPrice {
                return fmt.Errorf("min_price must be <= max_price")
        }
        if r.Radius > 0 && (r.Lat == nil || r.Lng == nil) {
                return fmt.Errorf("lat and lng required when radius is set")
        }
        return nil
}
