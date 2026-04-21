package resources

// APIResource represents the API service as a managed resource.
type APIResource struct {
	Replicas       int     `json:"replicas"`
	LatencySLO     float64 `json:"latency_slo_ms"` // P95 target
	AutoscaleMin   int     `json:"autoscale_min"`
	AutoscaleMax   int     `json:"autoscale_max"`
	CPUUtilization float64 `json:"cpu_utilization"`
	RPS            float64 `json:"rps"`
	Version        string  `json:"version"`
}

// DefaultAPIResource returns production defaults for the API resource.
func DefaultAPIResource() APIResource {
	return APIResource{
		Replicas:     4,
		LatencySLO:   300,
		AutoscaleMin: 2,
		AutoscaleMax: 10,
		Version:      "latest",
	}
}

// IsHealthy returns true if the API resource meets its SLO.
func (a *APIResource) IsHealthy() bool {
	return a.CPUUtilization < 80 && a.Replicas >= a.AutoscaleMin
}

// NeedsScaleUp returns true if the resource needs more replicas.
func (a *APIResource) NeedsScaleUp() bool {
	return a.CPUUtilization > 70 || a.RPS/float64(a.Replicas) > 50
}

// NeedsScaleDown returns true if the resource is overprovisioned.
func (a *APIResource) NeedsScaleDown() bool {
	return a.CPUUtilization < 30 && a.RPS < 50 && a.Replicas > a.AutoscaleMin
}
