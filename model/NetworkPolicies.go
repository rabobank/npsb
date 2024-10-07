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
	return fmt.Sprintf("%s (%s) => %s (%s:%d(%s))", np.SourceName, np.Source, np.DestinationName, np.Destination, np.Port, np.Protocol)
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

type InstancesWithBinds struct {
	BoundApps    []Destination `json:"bound_apps"`
	SrcOrDst     string        `json:"src_or_dst"`
	NameOrSource string        `json:"name_or_source"`
}

func (iwb InstancesWithBinds) String() string {
	var bindStr string
	for _, bind := range iwb.BoundApps {
		bindStr += fmt.Sprintf("%s:%d(%s), ", bind.Id, bind.Port, bind.Protocol)
	}
	return fmt.Sprintf("type:%s, name/source: %s, #binds: %d : %s", iwb.SrcOrDst, iwb.NameOrSource, len(iwb.BoundApps), bindStr)
}

type PolicyServerGetResponse struct {
	TotalPolicies int             `json:"total_policies"`
	Policies      []NetworkPolicy `json:"policies"`
}
