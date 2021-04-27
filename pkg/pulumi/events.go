package pulumi

import (
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	log "github.com/sirupsen/logrus"
)

func CollectEvents(eventChannel <-chan events.EngineEvent) {

	for {

		var event events.EngineEvent
		var ok bool

		createLogger := log.WithFields(log.Fields{"event": "CREATING"})
		completeLogger := log.WithFields(log.Fields{"event": "COMPLETE"})

		event, ok = <-eventChannel
		if !ok {
			return
		}

		if event.ResourcePreEvent != nil {

			switch event.ResourcePreEvent.Metadata.Type {
			case "aws:ecr/repository:Repository":
				createLogger.WithFields(log.Fields{"resource": event.ResourcePreEvent.Metadata.Type}).Info("Creating ECR repository")
			case "kubernetes:core/v1:Namespace":
				createLogger.WithFields(log.Fields{"resource": event.ResourcePreEvent.Metadata.Type}).Info("Creating Kubernetes Namespace")
			case "kubernetes:core/v1:Service":
				createLogger.WithFields(log.Fields{"resource": event.ResourcePreEvent.Metadata.Type}).Info("Creating Kubernetes Service")
			case "kubernetes:apps/v1:Deployment":
				createLogger.WithFields(log.Fields{"resource": event.ResourcePreEvent.Metadata.Type}).Info("Creating Kubernetes Deployment")
			case "docker:image:Image":
				createLogger.WithFields(log.Fields{"resource": event.ResourcePreEvent.Metadata.Type}).Info("Creating Docker Image")
			}
		}

		if event.ResOutputsEvent != nil {
			switch event.ResOutputsEvent.Metadata.Type {
			case "aws:ecr/repository:Repository":
				completeLogger.WithFields(log.Fields{"name": event.ResOutputsEvent.Metadata.New.Outputs["repositoryUrl"], "resource": event.ResOutputsEvent.Metadata.Type}).Info("Created ECR repository")
			case "kubernetes:core/v1:Namespace":
				completeLogger.WithFields(log.Fields{"resource": event.ResOutputsEvent.Metadata.Type}).Info("Created Kubernetes Namespace")
			case "kubernetes:core/v1:Service":
				completeLogger.WithFields(log.Fields{"resource": event.ResOutputsEvent.Metadata.Type}).Info("Created Kubernetes Service")
			case "kubernetes:apps/v1:Deployment":
				completeLogger.WithFields(log.Fields{"resource": event.ResOutputsEvent.Metadata.Type}).Info("Created Kubernetes Deployment")
			case "docker:image:Image":
				completeLogger.WithFields(log.Fields{"name": event.ResOutputsEvent.Metadata.New.Outputs["baseImageName"], "resource": event.ResOutputsEvent.Metadata.Type}).Info("Created Docker Image")
			}

		}
	}
}
