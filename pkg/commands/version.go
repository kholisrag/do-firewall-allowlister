package commands

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// VersionInfo holds version information
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	BuiltBy   string `json:"builtBy"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

// NewVersionCommand creates and returns the version command
func NewVersionCommand() *cobra.Command {
	var versionOutput string

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print detailed version information including build details",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(versionOutput)
		},
	}

	versionCmd.Flags().StringVarP(&versionOutput, "output", "o", "text", "Output format (text, json)")
	return versionCmd
}

func runVersion(output string) error {
	versionInfo := VersionInfo{
		Version:   "dev", // Will be set by build process
		Commit:    "none",
		Date:      "unknown",
		BuiltBy:   "unknown",
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	switch output {
	case "json":
		jsonOutput, err := json.MarshalIndent(versionInfo, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal version info: %w", err)
		}
		fmt.Println(string(jsonOutput))
	case "text":
		fmt.Printf("do-firewall-allowlister version %s\n", versionInfo.Version)
		fmt.Printf("  commit: %s\n", versionInfo.Commit)
		fmt.Printf("  built: %s\n", versionInfo.Date)
		fmt.Printf("  built by: %s\n", versionInfo.BuiltBy)
		fmt.Printf("  go version: %s\n", versionInfo.GoVersion)
		fmt.Printf("  platform: %s\n", versionInfo.Platform)
	default:
		return fmt.Errorf("unsupported output format: %s (supported: text, json)", output)
	}

	return nil
}
