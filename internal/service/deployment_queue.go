package service

import (
	"context"
	"encoding/json"
	"fmt"

	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"

	"gitlab.unanet.io/devops/eve/internal/data"
	"gitlab.unanet.io/devops/eve/pkg/errors"
	"gitlab.unanet.io/devops/eve/pkg/eve"
	json2 "gitlab.unanet.io/devops/eve/pkg/json"
	"gitlab.unanet.io/devops/eve/pkg/queue"
	"gitlab.unanet.io/devops/eve/pkg/s3"
)

type DeploymentQueueRepo interface {
	UpdateDeploymentReceiptHandle(ctx context.Context, id uuid.UUID, receiptHandle string) (*data.Deployment, error)
	DeployedServicesByNamespaceID(ctx context.Context, namespaceID int) (data.Services, error)
	DeployedDatabaseInstancesByNamespaceID(ctx context.Context, namespaceID int) (data.DatabaseInstances, error)
	UpdateDeploymentS3PlanLocation(ctx context.Context, id uuid.UUID, location json2.Text) error
	UpdateDeploymentS3ResultLocation(ctx context.Context, id uuid.UUID, location json2.Text) (*data.Deployment, error)
	ClusterByID(ctx context.Context, id int) (*data.Cluster, error)
}

type CloudUploader interface {
	Upload(ctx context.Context, key string, body []byte) (*s3.Location, error)
}

// API Queue Commands
const (
	CommandScheduleDeployment string = "api-schedule-deployment"
	CommandUpdateDeployment   string = "api-update-deployment"
)

// Scheduler Queue Commands
const (
	CommandDeployNamespace string = "sch-deploy-namespace"
)

type HttpCallback interface {
	Post(ctx context.Context, url string, body interface{}) error
}

func fromDataService(s data.Service) *eve.DeployService {
	return &eve.DeployService{
		ServiceID: s.ServiceID,
		DeployArtifact: &eve.DeployArtifact{
			ArtifactID:       s.ArtifactID,
			ArtifactName:     s.ArtifactName,
			RequestedVersion: s.RequestedVersion,
			DeployedVersion:  s.DeployedVersion.String,
			Metadata:         s.Metadata.AsMap(),
			Result:           eve.DeployArtifactResultNoop,
		},
	}
}

func fromDataServices(services data.Services) eve.DeployServices {
	var list eve.DeployServices
	for _, x := range services {
		list = append(list, fromDataService(x))
	}
	return list
}

func fromDataDatabaseInstance(s data.DatabaseInstance) *eve.DeployMigration {
	return &eve.DeployMigration{
		DatabaseID:   s.DatabaseID,
		DatabaseName: s.DatabaseName,
		DeployArtifact: &eve.DeployArtifact{
			ArtifactID:       s.ArtifactID,
			ArtifactName:     s.ArtifactName,
			RequestedVersion: s.RequestedVersion,
			DeployedVersion:  s.DeployedVersion.String,
			Metadata:         s.Metadata.AsMap(),
		},
	}
}

func fromDataDatabaseInstances(d data.DatabaseInstances) eve.DeployMigrations {
	var list eve.DeployMigrations
	for _, x := range d {
		list = append(list, fromDataDatabaseInstance(x))
	}
	return list
}

type messageLogger func(format string, a ...interface{})

type DeploymentQueue struct {
	worker   QueueWorker
	repo     DeploymentQueueRepo
	uploader CloudUploader
	callback HttpCallback
}

func NewDeploymentQueue(
	worker QueueWorker,
	repo DeploymentQueueRepo,
	uploader CloudUploader,
	httpCallBack HttpCallback) *DeploymentQueue {
	return &DeploymentQueue{
		worker:   worker,
		repo:     repo,
		uploader: uploader,
		callback: httpCallBack,
	}
}

func (dq *DeploymentQueue) Logger(ctx context.Context) *zap.Logger {
	return queue.GetLogger(ctx)
}

func (dq *DeploymentQueue) Start() {
	go func() {
		dq.worker.Start(queue.HandlerFunc(dq.handleMessage))
	}()
}

func (dq *DeploymentQueue) Stop() {
	dq.worker.Stop()
}

func (dq *DeploymentQueue) matchArtifact(a *eve.DeployArtifact, options NamespacePlanOptions, logger messageLogger) {
	// match services to be deployed
	match := options.Artifacts.Match(a.ArtifactID, a.RequestedVersion)
	if match == nil {
		return
	}
	match.Matched = true
	if a.DeployedVersion == match.AvailableVersion && !options.ForceDeploy {
		if options.ArtifactsSupplied {
			logger("artifact: %s, version: %s, is already up to date", a.ArtifactName, a.DeployedVersion)
		}
		return
	}
	a.AvailableVersion = match.AvailableVersion
	a.ArtifactoryPath = match.ArtifactoryPath
	a.ArtifactoryFeed = match.ArtifactoryFeed
	if a.AvailableVersion == "" || (a.DeployedVersion == a.AvailableVersion && !options.ForceDeploy) {
		return
	}
	a.Deploy = true
}

func (dq *DeploymentQueue) setupNSDeploymentPlan(ctx context.Context, options NamespacePlanOptions) (*eve.NSDeploymentPlan, error) {
	cluster, err := dq.repo.ClusterByID(ctx, options.NamespaceRequest.ClusterID)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	plan := eve.NSDeploymentPlan{
		Namespace:       options.NamespaceRequest,
		EnvironmentName: options.EnvironmentName,
		CallbackUrl:     options.CallbackURL,
		SchQueueUrl:     cluster.SchQueueUrl,
	}

	if options.DryRun == true {
		plan.Status = eve.DeploymentPlanStatusDryrun
	} else {
		plan.Status = eve.DeploymentPlanStatusPending
	}

	return &plan, nil
}

func (dq *DeploymentQueue) createServicesDeployment(ctx context.Context, options NamespacePlanOptions) (*eve.NSDeploymentPlan, error) {
	nSDeploymentPlan, err := dq.setupNSDeploymentPlan(ctx, options)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	dataServices, err := dq.repo.DeployedServicesByNamespaceID(ctx, options.NamespaceRequest.ID)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	services := fromDataServices(dataServices)
	for _, x := range services {
		dq.matchArtifact(x.DeployArtifact, options, nSDeploymentPlan.Message)
	}
	if options.ArtifactsSupplied {
		unmatched := options.Artifacts.UnMatched()
		for _, x := range unmatched {
			nSDeploymentPlan.Message("unmatched service: %s", x.Name)
		}
	}
	nSDeploymentPlan.Services = services.ToDeploy()
	return nSDeploymentPlan, nil
}

func (dq *DeploymentQueue) createMigrationsDeployment(ctx context.Context, options NamespacePlanOptions) (*eve.NSDeploymentPlan, error) {
	nSDeploymentPlan, err := dq.setupNSDeploymentPlan(ctx, options)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	dataDatabaseInstances, err := dq.repo.DeployedDatabaseInstancesByNamespaceID(ctx, options.NamespaceRequest.ID)
	if err != nil {
		return nil, errors.Wrap(err)
	}
	migrations := fromDataDatabaseInstances(dataDatabaseInstances)
	for _, x := range migrations {
		dq.matchArtifact(x.DeployArtifact, options, nSDeploymentPlan.Message)
	}
	if options.ArtifactsSupplied {
		unmatched := options.Artifacts.UnMatched()
		for _, x := range unmatched {
			nSDeploymentPlan.Message("unmatched service: %s", x.Name)
		}
	}
	nSDeploymentPlan.Migrations = migrations.ToDeploy()
	return nSDeploymentPlan, nil
}

func (dq *DeploymentQueue) rollbackError(ctx context.Context, m *queue.M, err error) error {
	qerr := dq.worker.DeleteMessage(ctx, m)
	if qerr != nil {
		dq.Logger(ctx).Error("an error occurred while trying to remove the message due to an error", zap.Any("queue_message", m), zap.Error(qerr))
	}
	return errors.Wrap(err)
}

func (dq *DeploymentQueue) scheduleDeployment(ctx context.Context, m *queue.M) error {
	deployment, err := dq.repo.UpdateDeploymentReceiptHandle(ctx, m.ID, m.ReceiptHandle)
	if err != nil {
		return dq.rollbackError(ctx, m, err)
	}

	var options NamespacePlanOptions
	err = json.Unmarshal(deployment.PlanOptions, &options)
	if err != nil {
		return dq.rollbackError(ctx, m, err)
	}

	var nsDeploymentPlan *eve.NSDeploymentPlan
	if options.Type == DeploymentPlanTypeApplication {
		nsDeploymentPlan, err = dq.createServicesDeployment(ctx, options)
	} else {
		nsDeploymentPlan, err = dq.createMigrationsDeployment(ctx, options)
	}
	if err != nil {
		return dq.rollbackError(ctx, m, err)
	}

	nsDeploymentPlanText, err := json2.StructToJson(nsDeploymentPlan)
	if err != nil {
		return dq.rollbackError(ctx, m, err)
	}

	if len(options.CallbackURL) > 0 {
		err := dq.callback.Post(ctx, options.CallbackURL, nsDeploymentPlan)
		if err != nil {
			dq.Logger(ctx).Warn("callback failed", zap.String("callback_url", options.CallbackURL))
		}
	}

	if options.DryRun {
		err = dq.worker.DeleteMessage(ctx, m)
		if err != nil {
			return dq.rollbackError(ctx, m, err)
		}
		return nil
	}

	location, err := dq.uploader.Upload(ctx, fmt.Sprintf("%s-plan", deployment.ID), nsDeploymentPlanText)
	if err != nil {
		return dq.rollbackError(ctx, m, err)
	}

	locationJson, err := json2.StructToJson(&location)
	if err != nil {
		return dq.rollbackError(ctx, m, err)
	}

	err = dq.worker.Message(ctx, nsDeploymentPlan.SchQueueUrl, &queue.M{
		ID:      deployment.ID,
		ReqID:   queue.GetReqID(ctx),
		GroupID: nsDeploymentPlan.GroupID(),
		Body:    locationJson,
		Command: CommandDeployNamespace,
	})
	if err != nil {
		return dq.rollbackError(ctx, m, err)
	}

	err = dq.repo.UpdateDeploymentS3PlanLocation(ctx, deployment.ID, locationJson)
	if err != nil {
		return dq.rollbackError(ctx, m, err)
	}

	return nil
}

func (dq *DeploymentQueue) handleMessage(ctx context.Context, m *queue.M) error {
	switch m.Command {
	// This means it hasn't been sent to the scheduler yet
	case CommandScheduleDeployment:
		return dq.scheduleDeployment(ctx, m)

	// This means it came back from the scheduler
	case CommandUpdateDeployment:
		return dq.updateDeployment(ctx, m)

	default:
		return errors.Wrapf("unrecognized command: %s", m.Command)
	}
}

func (dq *DeploymentQueue) updateDeployment(ctx context.Context, m *queue.M) error {
	deployment, err := dq.repo.UpdateDeploymentS3ResultLocation(ctx, m.ID, m.Body)
	if err != nil {
		return errors.Wrap(err)
	}

	// TODO: Update successful deployed db values in the database.

	// Here we are deleting the original deploy message which unblocks deployments for a namespace in an environment
	// We will need to add some additional logic to this to account for certain scenarios where we should
	// Still Delete the Message that triggers this updateDeployment (like an error that returns not found or already deleted)
	err = dq.worker.DeleteMessage(ctx, &queue.M{
		ID:            deployment.ID,
		ReqID:         queue.GetReqID(ctx),
		ReceiptHandle: deployment.ReceiptHandle.String,
	})
	if err != nil {
		return errors.Wrap(err)
	}

	err = dq.worker.DeleteMessage(ctx, m)
	if err != nil {
		return errors.Wrap(err)
	}

	//if len(options.CallbackURL) > 0 {
	//	err := dq.callback.Post(ctx, options.CallbackURL)
	//	if err != nil {
	//		dq.Logger(ctx).Warn("callback failed", zap.String("callback_url", options.CallbackURL))
	//	}
	//}

	return nil
}
