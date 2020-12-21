package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"go.uber.org/zap"

	"gitlab.unanet.io/devops/eve/internal/api"
	"gitlab.unanet.io/devops/eve/internal/data"
	"gitlab.unanet.io/devops/eve/internal/service/crud"
	"gitlab.unanet.io/devops/eve/internal/service/plans"
	"gitlab.unanet.io/devops/eve/internal/service/releases"
	"gitlab.unanet.io/devops/eve/pkg/artifactory"
	"gitlab.unanet.io/devops/eve/pkg/gitlab"
	"gitlab.unanet.io/devops/eve/pkg/queue"
	"gitlab.unanet.io/devops/eve/pkg/s3"
	"gitlab.unanet.io/devops/go/pkg/log"
)

func main() {
	dbConfig := api.GetDBConfig()
	// Try to get a DB Connection
	db, err := data.GetDBWithTimeout(dbConfig.DbConnectionString(), dbConfig.DBConnectionTimeout)
	if err != nil {
		log.Logger.Panic("Failed to open Connection to DB.", zap.Error(err))
	}

	flags := api.GetFlagsConfig()

	if flags.MigrateFlag {
		err = data.MigrateDB(dbConfig.MigrationConnectionString(), dbConfig.LogLevel)
		if err != nil {
			log.Logger.Panic("Failed to load the Database Migration Tool.", zap.Error(err))
		}
	}

	if !flags.ServerFlag {
		return
	}

	config := api.GetConfig()

	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AWSRegion)},
	)
	if err != nil {
		log.Logger.Panic("Failed to create AWS Session", zap.Error(err))
	}
	apiQueue := queue.NewQ(awsSession, queue.Config{
		MaxNumberOfMessage: config.ApiQMaxNumberOfMessage,
		QueueURL:           config.ApiQUrl,
		WaitTimeSecond:     config.ApiQWaitTimeSecond,
		VisibilityTimeout:  config.ApiQVisibilityTimeout,
	})

	repo := data.NewRepo(db)
	artifactoryClient := artifactory.NewClient(config.ArtifactoryConfig)
	deploymentPlanGenerator := plans.NewPlanGenerator(repo, artifactoryClient, apiQueue)
	crudManager := crud.NewManager(repo)
	gitlabClient := gitlab.NewClient(config.GitlabConfig)
	releaseSvc := releases.NewReleaseSvc(repo, artifactoryClient, gitlabClient)

	controllers, err := api.InitializeControllers(deploymentPlanGenerator, crudManager, releaseSvc)
	if err != nil {
		log.Logger.Panic("Unable to Initialize the Controllers")
	}
	apiServer, err := api.NewApi(controllers, config)
	if err != nil {
		log.Logger.Panic("Failed to Create Api App", zap.Error(err))
	}

	deploymentQueue := plans.NewQueue(
		queue.NewWorker("eve-api", apiQueue, config.ApiQWorkerTimeout),
		repo,
		crudManager,
		s3.NewUploader(awsSession, s3.Config{Bucket: config.S3Bucket}),
		s3.NewDownloader(awsSession),
		plans.NewCallback(config.HttpCallbackTimeout),
	)

	cron := plans.NewDeploymentCron(repo, deploymentPlanGenerator, config.CronTimeout)
	cron.Start()
	deploymentQueue.Start()

	apiServer.Start(func() {
		cron.Stop()
		deploymentQueue.Stop()
	})
}
