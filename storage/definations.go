package storage

type ProjectData struct {
	Key                string `json:"key"`
	ProjectName        []byte `json:"name"`
	ProjectDescription []byte `json:"description"`
	ProjectOwner       []byte `json:"owner"`
	Logo               []byte `json:"url"`
}

type UpdateData struct {
	Key                  string `json:"key"`
	ProjectTxID          []byte `json:"project_id"` // reference to Project
	UpdateExecutableHash []byte `json:"executable_hash"`
	UpdateIPFSUrl        []byte `json:"executable_ipfs_url"`
	ForDeviceName        []byte `json:"for_device_name"`
	UpdateVersion        uint8  `json:"version"`
	SuccessCount         uint8  `json:"success_count"`
}
