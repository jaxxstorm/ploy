package up

import (
	"context"
	"fmt"
	"os"

	n "github.com/jaxxstorm/ploy/pkg/name"
	pulumi "github.com/jaxxstorm/ploy/pkg/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dryrun    bool
	name      string
	directory string
	verbose   bool
	nlb       bool
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "up",
		Short: "Deploy your application",
		Long:  "Deploy your application to Kubernetes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			// Set some required params
			ctx := context.Background()
			org := viper.GetString("org")
			region := viper.GetString("region")

			if org == "" {
				return fmt.Errorf("must specify pulumi org via flag or config file")
			}

			// If the user doesn't specify a name, generate a random one for them
			if len(args) < 1 {
				name = n.GenerateName()
			} else {
				name = args[0]
			}

			// check if we have a valid Dockerfile before proceeding
			dockerfile := fmt.Sprintf("%s/Dockerfile", directory)
			if _, err := os.Stat(dockerfile); os.IsNotExist(err) {
				return fmt.Errorf("no Dockerfile found in %s: %v", directory, err)
			}

			// Create a stack in our backend
			// We place all apps we deploy in the same project, so we can list them later
			// Each app is a stack, so we can do this multiple times
			stackName := auto.FullyQualifiedStackName(org, "ploy", name)
			// Create a stack. We'll set the program shortly
			pulumiStack, err := auto.UpsertStackInlineSource(ctx, stackName, "ploy", nil)
			if err != nil {
				return fmt.Errorf("failed to create or select stack: %v", err)
			}

			// set the AWS region from config
			err = pulumiStack.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: region})
			if err != nil {
				return err
			}

			err = pulumiStack.SetConfig(ctx, "aws:skipMetadataApiCheck", auto.ConfigValue{Value: "false"})
			if err != nil {
				return err
			}

			// Set up the workspace and install all the required plugins the user needs
			workspace := pulumiStack.Workspace()

			err = pulumi.EnsurePlugins(workspace)

			if err != nil {
				return err
			}

			// Now, we set the pulumi program that is going to run
			workspace.SetProgram(pulumi.Deploy(name, directory, nlb))

			if dryrun {
				_, err = pulumiStack.Preview(ctx, optpreview.Message("Running ploy dryrun"))
				if err != nil {
					return fmt.Errorf("error creating stack: %v", err)
				}
			} else {
				// Wire up our update to stream progress to stdout
				// We give the user the option to actually view the Pulumi output
				var streamer optup.Option
				if verbose {
					streamer = optup.ProgressStreams(os.Stdout)
				} else {
					upChannel := make(chan events.EngineEvent)
					go collectEvents(upChannel)

					streamer = optup.EventStreams(upChannel)

				}
				_, err = pulumiStack.Up(ctx, streamer)

				if err != nil {
					return err
				}
			}

			return nil

		},
	}

	f := command.Flags()
	f.BoolVarP(&dryrun, "preview", "p", false, "Preview changes, dry-run mode")
	f.BoolVarP(&verbose, "verbose", "v", false, "Show output of Pulumi operations")
	f.StringVarP(&directory, "dir", "d", ".", "Path to docker context to use")
	f.BoolVar(&nlb, "nlb", false, "Provision an NLB instead of ELB")

	return command
}

func collectEvents(eventChannel <-chan events.EngineEvent) {
	for {
		event, ok := <-eventChannel
		if !ok {
			return
		}

		if event.ResourcePreEvent != nil {

			switch event.ResourcePreEvent.Metadata.Type {
			case "aws:ecr/repository:Repository":
				log.Infof("Creating ECR Repository")
			case "kubernetes:core/v1:Namespace":
				log.Infof("Creating Kubernetes Namespace")
			case "kubernetes:core/v1:Service":
				log.Infof("Creating Kubernetes Service")
			case "kubernetes:apps/v1:Deployment":
				log.Infof("Creating Kubernetes Deployment")
			case "docker:image:Image":
				log.Infof("Creating Docker Image")
			}
		}

		if event.ResOutputsEvent != nil {
			switch event.ResOutputsEvent.Metadata.Type {
			case "aws:ecr/repository:Repository":
				log.Infof("Created ECR Repository: %v", event.ResOutputsEvent.Metadata.New.Outputs["repositoryUrl"])
			case "kubernetes:core/v1:Namespace":
				log.Infof("Created Kubernetes Namespace")
			case "kubernetes:core/v1:Service":
				log.Infof("Created Kubernetes Service")
			case "kubernetes:apps/v1:Deployment":
				log.Infof("Created Kubernetes Deployment")
			case "docker:image:Image":
				log.Infof("Creating Docker Image: %v", event.ResOutputsEvent.Metadata.New.Outputs["baseImageName"])
			}

		}
	}
}
