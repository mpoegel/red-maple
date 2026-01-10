package homeassistant

import "time"

type DeviceState struct {
	EntityID   string `json:"entity_id"`
	State      string `json:"state"`
	Attributes struct {
		StateClass   string `json:"state_class"`
		Unit         string `json:"unit_of_measurement"`
		FriendlyName string `json:"friendly_name"`
	} `json:"attributes"`
	LastChanged  time.Time `json:"last_changed"`
	LastReported time.Time `json:"last_reported"`
	LastUpdated  time.Time `json:"last_updated"`
	Context      struct {
		ID       string `json:"id"`
		ParentID string `json:"parent_id"`
		UserID   string `json:"user_id"`
	} `json:"context"`
}
