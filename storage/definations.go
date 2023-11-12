package storage

type ProjectData struct {
	Key                string `json:"key"`
	ProjectName        []byte `json:"name"`
	ProjectDescription []byte `json:"description"`
	Owner              []byte `json:"owner"`
	Logo               []byte `json:"url"`
}
