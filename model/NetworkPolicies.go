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

type NetworkPolicies struct {
	Policies []NetworkPolicy `json:"policies"`
}

type NetworkPolicy struct {
	Source      Source      `json:"source"`
	Destination Destination `json:"destination"`
}

type Source struct {
	Id string `json:"id"`
}

type Destination struct {
	Id       string `json:"id"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
}
