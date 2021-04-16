package program

import (
	"context"

	"fmt"

	_ "github.com/pulumi/pulumi/sdk/v2/go/common/workspace"
	"github.com/pulumi/pulumi/sdk/v2/go/x/auto"
)

// EnsurePlugins installs the needed Pulumi plugins for ploy to run
func EnsurePlugins(workspace auto.Workspace) error {

	ctx := context.Background()

	err := workspace.InstallPlugin(ctx, "aws", "v3.11.0")
	if err != nil {
		return fmt.Errorf("error installing aws plugin: %v", err)
	}
	err = workspace.InstallPlugin(ctx, "kubernetes", "v2.6.3")
	if err != nil {
		return fmt.Errorf("error installing kubernetes plugin: %v", err)
	}
	err = workspace.InstallPlugin(ctx, "docker", "v2.4.0")
	if err != nil {
		return fmt.Errorf("error installing docker plugin: %v", err)
	}

	return nil

}
