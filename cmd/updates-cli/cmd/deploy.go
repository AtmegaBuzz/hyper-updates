package cmd

import (
	"context"
	"fmt"
	"hyper-updates/actions"

	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use: "deploy",
	RunE: func(*cobra.Command, []string) error {
		return ErrMissingSubcommand
	},
}

// struct {
//  repository
// 	repository_name:
// 	owner: address
//  project_description:
// 	logo: "ipfs url"
// }

// struct {
// 	repository_id:
// 	version_name:
// 	version_url:
// 	hash:
// }

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
		// Path, err := handler.Root().PromptString("Project Logo", 1, 1000)
		// if err != nil {
		// 	return err
		// }

		// URL, err := deployPinata(
		// 	Path,
		// 	"fc43a725fd778580045c",
		// 	"37c52b3571d7df2c1326c1460a1b192c209a1fb212c6b1b96eb2626bb2076efe",
		// )
		URL := "https://upload.wikimedia.org/wikipedia/commons/thumb/2/2f/Google_2015_logo.svg/1200px-Google_2015_logo.svg.png"

		if err != nil {
			return err
		}

		// Add project description to project
		project_description, err := handler.Root().PromptString("Project Description", 1, actions.ProjectDescriptionUnits)
		if err != nil {
			return err
		}

		// get current auth user
		_, priv, _, _, _, _, err := handler.DefaultActor()
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
			Owner:              priv.Address,
			Logo:               []byte(URL),
		}

		// Generate transaction
		_, id, err := sendAndWait(ctx, nil, project, cli, scli, tcli, factory, true)

		fmt.Println(id)

		return err

	},
}

// var deploycodeCmd = &cobra.Command{
// 	Use: "deploy-code",
// 	RunE: func(*cobra.Command, []string) error {

// 		ctx := context.Background()
// 		_, _, factory, cli, scli, tcli, err := handler.DefaultActor()
// 		if err != nil {
// 			return err
// 		}

// 		// Add symbol to token
// 		ID, err := handler.Root().PromptString("ID", 1, 256)
// 		if err != nil {
// 			return err
// 		}

// 		// Add decimal to token
// 		Path, err := handler.Root().PromptString("Asset Image Path", 1, 256)
// 		if err != nil {
// 			return err
// 		}

// 		URL, err := deployPinata(
// 			Path,
// 			"fc43a725fd778580045c",
// 			"37c52b3571d7df2c1326c1460a1b192c209a1fb212c6b1b96eb2626bb2076efe",
// 		)

// 		if err != nil {
// 			return err
// 		}

// 		// Add metadata to token
// 		metadata, err := handler.Root().PromptString("metadata", 1, actions.MaxMetadataSize)
// 		if err != nil {
// 			return err
// 		}

// 		Owner, err := handler.Root().PromptString("recipient", 1, 256)

// 		// Confirm action
// 		cont, err := handler.Root().PromptContinue()
// 		if !cont || err != nil {
// 			return err
// 		}

// 		nft := &actions.CreateNFT{
// 			ID:       []byte(ID),
// 			Metadata: []byte(metadata),
// 			Owner:    []byte(Owner),
// 			URL:      []byte(URL),
// 		}

// 		// Generate transaction
// 		_, _id, err := sendAndWait(ctx, nil, nft, cli, scli, tcli, factory, true)

// 		storage.StoreNFT(_id.String(), nft.ID, nft.Metadata, nft.Owner, nft.URL)

// 		return err
// 	},
// }
