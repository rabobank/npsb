package model

import (
	"fmt"
)

type NetworkPolicyLabels struct {
	Source          string `json:"source"` // app guid
	SourceName      string
	Destination     string `json:"destination"`
	DestinationName string
	Protocol        string `json:"protocol"`
	Port            int    `json:"port"`
}

func (np NetworkPolicyLabels) String() string {
	return fmt.Sprintf("%s (%s) => %s (%s):%d", np.SourceName, np.Source, np.DestinationName, np.Destination, np.Port)
}
