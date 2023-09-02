package knowledge

type KSType struct {
	Name     string `json:"name"`
	Solution string `json:"solution"`
}

type KSObject struct {
	ID   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}
