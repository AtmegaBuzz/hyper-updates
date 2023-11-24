package cmd

import (
	"context"
	"fmt"
	"hyper-updates/actions"
	"hyper-updates/consts"

	"github.com/ava-labs/hypersdk/codec"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use: "deploy",
	RunE: func(*cobra.Command, []string) error {
		return ErrMissingSubcommand
	},
}

var createRepoCmd = &cobra.Command{
	Use: "create-repository",
	RunE: func(*cobra.Command, []string) error {

		ctx := context.Background()
		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		// Ask Repository/storage name
		project_name, err := handler.Root().PromptString("Project Name", 1, 1000)
		if err != nil {
			return err
		}

		// Project logo path
		URL, err := handler.Root().PromptString("Project Logo URL", 1, 1000)
		if err != nil {
			return err
		}

		// Add project description to project
		project_description, err := handler.Root().PromptString("Project Description", 1, actions.ProjectDescriptionUnits)
		if err != nil {
			return err
		}

		// Confirm action
		cont, err := handler.Root().PromptContinue()
		if !cont || err != nil {
			return err
		}

		project := &actions.CreateProject{
			ProjectName:        []byte(project_name),
			ProjectDescription: []byte(project_description),
			Logo:               []byte(URL),
		}

		// Generate transaction
		_, id, err := sendAndWait(ctx, nil, project, cli, scli, tcli, factory, true)

		if err != nil {
			fmt.Println("Error occured")
		}

		fmt.Println(id)

		return err

	},
}

var getRepoCmd = &cobra.Command{
	Use: "get-repository",
	RunE: func(*cobra.Command, []string) error {

		ctx := context.Background()
		_, _, _, _, _, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		id, err := handler.Root().PromptID("Project txid")

		ID, ProjectName, ProjectDescription, ProjectOwner, Logo, err := tcli.Project(ctx, id, false)

		addr, err := codec.AddressBech32(consts.HRP, codec.Address(ID))
		// owner, err := codec.AddressBech32(consts.HRP, codec.Address(ProjectOwner))

		fmt.Println("Id: ", addr, ", Project Name: ", string(ProjectName), ", Project Logo: ", string(Logo), ", Project Description: ", string(ProjectDescription), ", Project Owner: ", string(ProjectOwner))

		return err

	},
}

var createUpdateCmd = &cobra.Command{
	Use: "push-update",
	RunE: func(*cobra.Command, []string) error {

		ctx := context.Background()
		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		project_id, err := handler.Root().PromptString("Project txid", 1, 100)
		if err != nil {
			return err
		}

		executable_path, err := handler.Root().PromptString("Executable Path", 1, 500)
		if err != nil {
			return err
		}

		executable_ipfs_url, err := DeployBin(
			executable_path,
			"fc43a725fd778580045c",
			"37c52b3571d7df2c1326c1460a1b192c209a1fb212c6b1b96eb2626bb2076efe",
		)
		if err != nil {
			return err
		}

		fmt.Println("Binary Upload completed")

		executable_hash, err := CalculateMD5(executable_path)
		if err != nil {
			return err
		}

		fmt.Println("Hash Calculated")

		for_device_name, err := handler.Root().PromptString("Update For Device (Name)", 1, 100)
		if err != nil {
			return err
		}

		version, err := handler.Root().PromptInt("Update Version", 10)
		if err != nil {
			return err
		}

		update := &actions.CreateUpdate{
			ProjectTxID:          []byte(project_id),
			UpdateExecutableHash: []byte(executable_hash),
			UpdateIPFSUrl:        []byte(executable_ipfs_url),
			ForDeviceName:        []byte(for_device_name),
			UpdateVersion:        uint8(version),
			SuccessCount:         0,
		}

		// Generate transaction
		_, id, err := sendAndWait(ctx, nil, update, cli, scli, tcli, factory, true)

		if err != nil {
			fmt.Println("Error occured while pushing the update")
		}

		fmt.Println(id)

		return err

	},
}

var getUpdateCmd = &cobra.Command{
	Use: "get-update",
	RunE: func(*cobra.Command, []string) error {

		ctx := context.Background()
		_, _, _, _, _, tcli, err := handler.DefaultActor()
		if err != nil {
			return err
		}

		id, err := handler.Root().PromptID("Update txid")

		ID, ProjectTxID, UpdateExecutableHash, UpdateIPFSUrl, ForDeviceName, UpdateVersion, SuccessCount, err := tcli.Update(ctx, id, false)

		addr, err := codec.AddressBech32(consts.HRP, codec.Address(ID))

		fmt.Println("Id: ", addr, ", Project Tx Id: ", string(ProjectTxID), ", Exe Hash: ", string(UpdateExecutableHash), ", Ipfs URL: ", string(UpdateIPFSUrl), ", For Devide: ", string(ForDeviceName), ", Version: ", UpdateVersion, ", Success: ", SuccessCount)

		return err

	},
}
