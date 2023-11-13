package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"hyper-updates/consts"

	"hyper-updates/version"
)

func init() {
	cobra.EnablePrefixMatching = true
}

// NewCommand implements "updatesvm version" command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Prints out the verson",
		RunE:  versionFunc,
	}
	return cmd
}

func versionFunc(*cobra.Command, []string) error {
	fmt.Printf("%s@%s (%s)\n", consts.Name, version.Version, consts.ID)
	return nil
}
