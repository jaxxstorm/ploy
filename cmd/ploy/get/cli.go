package get

import (
	"context"
	"fmt"
	"os"

	n "github.com/jaxxstorm/ploy/pkg/name"
	"github.com/olekukonko/tablewriter"
	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "get",
		Short: "Get all ploy deployed applications",
		Long:  "Get all ploy deployed applications",
		RunE: func(cmd *cobra.Command, args []string) error {

			// Required params
			ctx := context.Background()
			org := viper.GetString("org")

			if org == "" {
				return fmt.Errorf("must specify pulumi org via flag or config file")
			}

			// Generate a random name to use
			name := n.GenerateName()

			/*
			 * This is a little hacky, but it works.
			 * I don't want to have to manage the workspaces myself, so I generate a stack name randomly
			 */
			stackName := auto.FullyQualifiedStackName(org, "ploy", name)
			// Create a stack. We never set the program, we're not going to use it
			pulumiStack, err := auto.UpsertStackInlineSource(ctx, stackName, "ploy", nil)
			if err != nil {
				return fmt.Errorf("failed to create or select stack: %v\n", err)
			}

			workspace := pulumiStack.Workspace()

			// Then we delete the stack from earlier so we don't include it in our list
			workspace.RemoveStack(ctx, name)

			// List the stacks in our workspace, each stack is an instance of an app
			stackList, err := workspace.ListStacks(ctx)
			if err != nil {
				return fmt.Errorf("failed to list available stacks: %v\n", err)
			}

			if len(stackList) > 0 {

				// Build a pretty table!
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Name", "Last Update", "Deployment Info", "URL"})

				// Loop through all the values in the returned stacks and add them to a string array
				for _, values := range stackList {

					// loop through all the stacks to retrieve the stack outputs
					stackName := auto.FullyQualifiedStackName(org, "ploy", values.Name)
					stack, err := auto.SelectStack(ctx, stackName, workspace)
					if err != nil {
						return fmt.Errorf("error selecting stack")
					}
					out, err := stack.Outputs(ctx)

					var url string
					if out["address"].Value == nil {
						url = ""
					} else {
						url = fmt.Sprintf("http://%s", out["address"].Value.(string))
					}

					// add all the values to the output tables
					table.Append([]string{values.Name, values.LastUpdate, values.URL, url})
				}

				// Render the table to stdout
				table.Render()
			} else {
				log.Info("No ploy apps currently deployed")
			}

			return nil
		},
	}

	return command
}
