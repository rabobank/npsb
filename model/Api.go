package model

// SourcesResponseList is the response from the /api/sources endpoint
type SourcesResponseList struct {
	SourcesResponses []SourceResponse `json:"source_responses"`
}

type SourceResponse struct {
	Source      string `json:"source"`
	Org         string `json:"org"`
	Space       string `json:"space"`
	Description string `json:"description"`
}

// GenericRequest - a generic request object
type GenericRequest struct {
	SpaceGUID string `json:"spaceguid"`
}
