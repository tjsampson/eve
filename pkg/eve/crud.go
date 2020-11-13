package eve

import (
	"context"
	"errors"
	"time"
)

type Environment struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Alias       string    `json:"alias,omitempty"`
	Description string    `json:"description"`
	Metadata    Metadata  `json:"metadata,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Namespace struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	Alias              string    `json:"alias"`
	EnvironmentID      int       `json:"environment_id"`
	EnvironmentName    string    `json:"environment_name"`
	RequestedVersion   string    `json:"requested_version"`
	ExplicitDeployOnly bool      `json:"explicit_deploy_only"`
	ClusterID          int       `json:"cluster_id"`
	Metadata           Metadata  `json:"metadata,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type Service struct {
	ID              int       `json:"id"`
	NamespaceID     int       `json:"namespace_id"`
	NamespaceName   string    `json:"namespace_name"`
	ArtifactID      int       `json:"artifact_id"`
	ArtifactName    string    `json:"artifact_name"`
	OverrideVersion string    `json:"override_version"`
	DeployedVersion string    `json:"deployed_version"`
	Metadata        Metadata  `json:"metadata,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Name            string    `json:"name"`
	StickySessions  bool      `json:"sticky_sessions"`
	NodeGroup       string    `json:"node_group"`
	Count           int       `json:"count"`
}

type Metadata map[string]interface{}

func (m Metadata) ValidateWithContext(ctx context.Context) error {
	if m == nil {
		return nil
	}

	if _, ok := m[""]; ok {
		return errors.New("cannot have an empty value as a key for metadata")
	}

	return nil
}
