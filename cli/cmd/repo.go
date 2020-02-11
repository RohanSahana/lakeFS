/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"fmt"
	"github.com/treeverse/lakefs/api/gen/models"
	"github.com/treeverse/lakefs/uri"
	"os"
	"time"

	"github.com/go-openapi/swag"

	"github.com/jedib0t/go-pretty/table"

	"github.com/spf13/cobra"
)

const (
	DefaultBranch = "master"
)

// repoCmd represents the repo command
var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "manage and explore repos",
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "list repositories",
	RunE: func(cmd *cobra.Command, args []string) error {

		amount, _ := cmd.Flags().GetInt("amount")
		after, _ := cmd.Flags().GetString("after")

		clt, err := getClient()
		if err != nil {
			return err
		}

		repos, pagination, err := clt.ListRepositories(context.Background(), after, amount)
		// generate table
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Repository", "Creation Date", "Default Branch Name", "Storage Namespace"})
		for _, repo := range repos {
			ts := time.Unix(repo.CreationDate, 0).Format(time.RFC1123Z)
			t.AppendRow([]interface{}{repo.ID, ts, repo.DefaultBranch, repo.BucketName})
		}
		t.Render()

		if pagination != nil && swag.BoolValue(pagination.HasMore) {
			fmt.Printf("for more results, run with `--amount %d --after \"%s\"\n", amount, pagination.NextOffset)
		}

		return nil
	},
}

// repoCreateCmd represents the create repo command
// lakectl create lakefs://myrepo mybucket
// not verifying bucket name in order not to bind to s3
var repoCreateCmd = &cobra.Command{
	Use:   "create  [repository uri] [bucket name]",
	Short: "create a new repository ",
	Args: ValidationChain(
		HasNArgs(2),
		IsRepoURI(0),
	),

	RunE: func(cmd *cobra.Command, args []string) error {

		clt, err := getClient()
		if err != nil {
			return err
		}
		u := uri.Must(uri.Parse(args[0]))
		defaultBranch, err := cmd.Flags().GetString("default-branch")
		if err != nil {
			return err
		}
		err = clt.CreateRepository(context.Background(), &models.RepositoryCreation{
			BucketName:    &args[1],
			DefaultBranch: defaultBranch,
			ID:            &u.Repository,
		})
		if err != nil {
			return err
		}

		repo, err := clt.GetRepository(context.Background(), u.Repository)
		if err != nil {
			return err
		}
		fmt.Printf("Repository '%s' created:\nbucket name: %s\ndefault branch: %s\ntimestamp: %d\n",
			repo.ID, repo.BucketName, repo.DefaultBranch, repo.CreationDate)
		return nil
	},
}

// repoDeleteCmd represents the delete repo command
// lakectl delete lakefs://myrepo
var repoDeleteCmd = &cobra.Command{
	Use:   "delete [repository uri]",
	Short: "delete existing repository",
	Args: ValidationChain(
		HasNArgs(1),
		IsRepoURI(0),
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		clt, err := getClient()
		if err != nil {
			return err
		}
		u := uri.Must(uri.Parse(args[0]))
		confirmation, err := confirm(fmt.Sprintf("Are you sure you want to delete repository: %s", u.Repository))
		if err != nil || !confirmation {
			fmt.Printf("Delete Repository '%s' aborted:\n", u.Repository)
			return nil
		}
		err = clt.DeleteRepository(context.Background(), u.Repository)
		if err != nil {
			return err
		}
		fmt.Printf("Repository '%s' deleted:\n", u.Repository)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoCreateCmd)
	repoCmd.AddCommand(repoDeleteCmd)

	repoListCmd.Flags().Int("amount", -1, "how many results to return, or-1 for all results (used for pagination)")
	repoListCmd.Flags().String("after", "", "show results after this value (used for pagination)")

	repoCreateCmd.Flags().StringP("default-branch", "d", DefaultBranch, "the default branch of this repository")

}
