package cfutils

type Progress struct {
	AccountID       string            `json:"accountId"`
	HealthCheck     int64             `json:"healthCheck"`
	ID              string            `json:"id"`
	LastUpdate      int64             `json:"lastUpdate"`
	Status          string            `json:"status"`
	Steps           []Step            `json:"steps"`
	StepsReferences map[string]string `json:"stepsReferences"`
	Visibility      string            `json:"visibility"`
}

type Step struct {
	CreationTimeStamp *int64   `json:"creationTimeStamp,omitempty"`
	FinishTimeStamp   *int64   `json:"finishTimeStamp,omitempty"`
	Logs              []string `json:"logs"`
	Name              string   `json:"name"`
	Progress          *string  `json:"progress,omitempty"`
	Status            string   `json:"status"`
}