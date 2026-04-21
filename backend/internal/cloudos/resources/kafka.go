package resources

// KafkaResource represents the Kafka infrastructure as a managed resource.
type KafkaResource struct {
	Partitions    int     `json:"partitions"`
	ConsumerLag   int64   `json:"consumer_lag"`
	ThroughputMB  float64 `json:"throughput_mb"`
	Replicas      int     `json:"replicas"` // consumer group replicas
	RetentionHours int    `json:"retention_hours"`
}

// DefaultKafkaResource returns production defaults.
func DefaultKafkaResource() KafkaResource {
	return KafkaResource{
		Partitions:     6,
		ConsumerLag:    0,
		Replicas:       3,
		RetentionHours: 72,
	}
}

// IsHealthy returns true if Kafka is operating normally.
func (k *KafkaResource) IsHealthy() bool {
	return k.ConsumerLag < 5000
}

// NeedsScaleUp returns true if consumer lag is too high.
func (k *KafkaResource) NeedsScaleUp() bool {
	return k.ConsumerLag > 2000
}
