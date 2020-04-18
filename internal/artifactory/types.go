package artifactory

type VersionResponse struct {
	Version string `json:"version"`
}

type MessagesResponse struct {
	Messages []Message `json:"messages"`
}

type Message struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type MoveRequest struct {
	RepoKey       string `json:"repoKey"`
	Path          string `json:"path"`
	TargetRepoKey string `json:"targetRepoKey"`
	TargetPath    string `json:"targetPath"`
}

type Properties struct {
	Properties map[string][]string `json:"properties"`
	URI        string              `json:"uri"`
}

func (p Properties) Property(key string) string {
	if val, ok := p.Properties[key]; ok {
		if len(val) == 0 {
			return ""
		}

		return val[0]
	}
	return ""
}

type AQLResult struct {
	Results []struct {
		Path       string `json:"path"`
		Name       string `json:"name"`
		Properties []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"properties"`
	} `json:"results"`
}
