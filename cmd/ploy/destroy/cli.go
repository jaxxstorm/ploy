package destroy

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	program "github.com/jaxxstorm/ploy/pkg/pulumi"
	"github.com/manifoldco/promptui"
	"github.com/pulumi/pulumi/sdk/v2/go/x/auto"
	"github.com/pulumi/pulumi/sdk/v2/go/x/auto/optdestroy"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dryrun    bool
	name      string
	directory string
	verbose   bool
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "destroy",
		Short: "Remove your application",
		Long:  "Remove your application from Kubernetes",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			ctx := context.Background()
			org := viper.GetString("org")
			region := viper.GetString("region")
			name := args[0]

			if org == "" {
				return fmt.Errorf("must specify pulumi org via flag or config file")
			}

			label := fmt.Sprintf("This will delete the application %s. Are you sure you wish to continue?", name)

			prompt := promptui.Prompt{
				Label:     label,
				IsConfirm: true,
			}

			result, err := prompt.Run()

			if err != nil {
				fmt.Printf("User cancelled, not deleting %v\n", err)
				os.Exit(0)
			}

			log.Debug("User confirmed, continuing: %s", result)
			log.Infof("Deleting application: %s", name)

			// create a stack in our backend
			stackName := auto.FullyQualifiedStackName(org, "ploy", name)
			// create a stack. We'll set the program shortly
			pulumiStack, err := auto.UpsertStackInlineSource(ctx, stackName, "ploy", nil)
			if err != nil {
				return fmt.Errorf("failed to create or select stack: %v\n", err)
			}

			// set the AWS region from config
			err = pulumiStack.SetConfig(ctx, "aws:region", auto.ConfigValue{Value: region})
			if err != nil {
				return err
			}

			// set up workspace and install plugins
			workspace := pulumiStack.Workspace()

			err = program.EnsurePlugins(workspace)

			if err != nil {
				return err
			}

			var streamer optdestroy.Option
			if verbose {
				streamer = optdestroy.ProgressStreams(os.Stdout)
			} else {
				streamer = optdestroy.ProgressStreams(ioutil.Discard)
			}
			_, err = pulumiStack.Destroy(ctx, streamer)

			if err != nil {
				return fmt.Errorf("error deleting stack resources: %v", err)
			}

			// destroy the stack so it's no longer listed
			// Then we delete the stack from earlier so we don't include it in our list
			workspace.RemoveStack(ctx, name)

			return nil

		},
	}
	f := command.Flags()
	f.BoolVarP(&dryrun, "preview", "p", false, "Preview changes, dry-run mode")
	f.BoolVarP(&verbose, "verbose", "v", false, "Show output of Pulumi operations")
	f.StringVarP(&directory, "dir", "d", ".", "Path to docker context to use")

	viper.BindPFlag("stack", command.Flags().Lookup("stack"))

	cobra.MarkFlagRequired(f, "name")
	return command
}
