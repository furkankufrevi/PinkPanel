package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.4.0-alpha"

func main() {
	rootCmd := &cobra.Command{
		Use:   "pinkpanel-cli",
		Short: "PinkPanel CLI tool",
	}

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(statusCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("pinkpanel-cli %s\n", version)
		},
	}
}

func statusCmd() *cobra.Command {
	var serverURL string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check panel health status",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := http.Get(serverURL + "/api/health")
			if err != nil {
				return fmt.Errorf("failed to connect to panel: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("invalid response: %w", err)
			}

			prettyJSON, _ := json.MarshalIndent(result, "", "  ")
			fmt.Println(string(prettyJSON))
			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "url", "http://localhost:8443", "Panel server URL")
	return cmd
}
