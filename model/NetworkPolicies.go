package model

import (
	"fmt"
)

type NetworkPolicy struct {
	Source          string `json:"source"` // app guid
	SourceName      string
	Destination     string `json:"destination"`
	DestinationName string
	Protocol        string `json:"protocol"`
	Port            string `json:"port"`
}

func (np NetworkPolicy) String() string {
	return fmt.Sprintf("%s (%s) => %s (%s):%s", np.SourceName, np.Source, np.DestinationName, np.Destination, np.Port)
}
