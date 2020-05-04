package eve

import (
	"fmt"
)

type DeployArtifact struct {
	ArtifactID       int                    `json:"artifact_id"`
	ArtifactName     string                 `json:"artifact_name"`
	RequestedVersion string                 `json:"requested_version"`
	DeployedVersion  string                 `json:"deployed_version"`
	AvailableVersion string                 `json:"available_version"`
	Metadata         map[string]interface{} `json:"metadata"`
	ArtifactoryFeed  string                 `json:"artifactory_feed"`
	ArtifactoryPath  string                 `json:"artifactory_path"`
	Deploy           bool                   `json:"-"`
}

type DeployService struct {
	*DeployArtifact
	ServiceID int `json:"service_id"`
}

type DeployServices []*DeployService

func (ds DeployServices) ToDeploy() DeployServices {
	var list DeployServices
	for _, x := range ds {
		if x.Deploy {
			list = append(list, x)
		}
	}
	return list
}

type DeployMigration struct {
	*DeployArtifact
	DatabaseID   int    `json:"database_id"`
	DatabaseName string `json:"database_name"`
}

type DeployMigrations []*DeployMigration

func (ds DeployMigrations) ToDeploy() DeployMigrations {
	var list DeployMigrations
	for _, x := range ds {
		if x.Deploy {
			list = append(list, x)
		}
	}
	return list
}

type NamespaceRequest struct {
	ID        int    `json:"id"`
	Alias     string `json:"alias"`
	Name      string `json:"name"`
	ClusterID int    `json:"cluster_id"`
}

func (ns *NamespaceRequest) GetQueueGroupID() string {
	return fmt.Sprintf("deploy-%s", ns.Name)
}

type NamespaceRequests []*NamespaceRequest

func (n NamespaceRequests) ToIDs() []int {
	var ids []int
	for _, x := range n {
		ids = append(ids, x.ID)
	}
	return ids
}

type NSDeploymentPlan struct {
	Namespace       *NamespaceRequest `json:"namespace"`
	EnvironmentName string            `json:"environment_name"`
	Services        DeployServices    `json:"services,omitempty"`
	Migrations      DeployMigrations  `json:"migrations,omitempty"`
	Messages        []string          `json:"messages,omitempty"`
	SchQueueUrl     string            `json:"-"`
}

func (ns *NSDeploymentPlan) GroupID() string {
	return ns.Namespace.Name
}

func (ns *NSDeploymentPlan) Message(format string, a ...interface{}) {
	ns.Messages = append(ns.Messages, fmt.Sprintf(format, a...))
}
