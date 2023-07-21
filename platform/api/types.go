package api

type KSType struct {
	Name     string `json:"name"`
	Solution string `json:"solution"`
}

type KSObject struct {
	ID   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}

type KSSolution struct {
	ID             string `json:"id"`
	LayerID        string `json:"layerId"`
	LayerType      string `json:"layerType"`
	ObjectMimeType string `json:"objectMimeType"`
	TargetObjectId string `json:"targetObjectId"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	DisplayName    string `json:"displayName"`
}

type KSItem interface {
	any | KSType | KSObject | KSSolution
}

type KSCollectionResponse[T KSItem] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}
