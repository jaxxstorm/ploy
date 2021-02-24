package up

import (
	"context"
	"fmt"
	"github.com/fatih/color"
	n "github.com/jaxxstorm/ploy/pkg/name"
	program "github.com/jaxxstorm/ploy/pkg/pulumi"
	"github.com/jaxxstorm/ploy/pkg/utils"
	"github.com/pulumi/pulumi/sdk/v2/go/x/auto"
	"github.com/pulumi/pulumi/sdk/v2/go/x/auto/optpreview"
	"github.com/pulumi/pulumi/sdk/v2/go/x/auto/optup"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
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
				return fmt.Errorf("no Dockerfile found in %s: %v\n", directory, err)
			}

			// Create a stack in our backend
			// We place all apps we deploy in the same project, so we can list them later
			// Each app is a stack, so we can do this multiple times
			stackName := auto.FullyQualifiedStackName(org, "ploy", name)
			// Create a stack. We'll set the program shortly
			pulumiStack, err := auto.UpsertStackInlineSource(ctx, stackName, "ploy", nil)
			if err != nil {
				return fmt.Errorf("failed to create or select stack: %v\n", err)
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
			err = workspace.InstallPlugin(ctx, "aws", "v3.30.0")
			if err != nil {
				return fmt.Errorf("error installing aws plugin: %v\n", err)
			}
			err = workspace.InstallPlugin(ctx, "kubernetes", "v2.8.2")
			if err != nil {
				return fmt.Errorf("error installing kubernetes plugin: %v\n", err)
			}
			err = workspace.InstallPlugin(ctx, "docker", "v2.8.1")
			if err != nil {
				return fmt.Errorf("error installing docker plugin: %v\n", err)
			}

			// Now, we set the pulumi program that is going to run
			workspace.SetProgram(program.Deploy(name, directory, nlb))

			if dryrun {
				_, err = pulumiStack.Preview(ctx, optpreview.Message("Running ploy dryrun"))
				if err != nil {
					return fmt.Errorf("error creating stack: %v\n", err)
				}
			} else {

				actionSpinner := utils.TerminalSpinner{
					SpinnerText:   fmt.Sprintf("Creating ploy deployment: %s", name),
					CompletedText: fmt.Sprintf("✅ Ploy deployment created: %s", name),
					FailureText:   fmt.Sprintf("❌ Error creating Ploy deployment: %s", name),
				}

				actionSpinner.Create()
				actionSpinner.SetOutput(os.Stdout)
				
				outputLogger := utils.TerminalSpinnerLogger{}
				// Stream the results to the terminal.
				streamer := optup.ProgressStreams(&outputLogger)

				// Wire up our update to stream progress to stdout
				// We give the user the option to actually view the Pulumi output
				if verbose {
					streamer = optup.ProgressStreams(&outputLogger)
				} else {
					streamer = optup.ProgressStreams(ioutil.Discard)
				}
				result, err := pulumiStack.Up(ctx, streamer)

				if err != nil {
					actionSpinner.Fail()
					return fmt.Errorf("error running stack update: %v", err)
				}

				actionSpinner.Stop()

				utils.ClearLine()
				utils.Print("")

				utils.Print(utils.TextColor("Outputs", color.FgGreen))
				for key, value := range result.Outputs {
					utils.Printf("    - %s: %v\n", key, value.Value)
				}

				utils.Print("")
				return nil

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
