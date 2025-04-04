package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	
	"driftdetector/internal/orchestrator"
)

func main() {
	fmt.Println("Drift Detector starting...")

	var instanceIDs string
	var configPath string
	var attributesToCheck string
	var outputFormat string
	var concurrencyLimit int

	rootCmd := &cobra.Command{
		Use:   "driftdetector",
		Short: "Detect infrastructure drift between AWS EC2 instances and Terraform configurations",
		Run: func(cmd *cobra.Command, args []string) {
			// Check required flags
			if instanceIDs == "" || configPath == "" {
				fmt.Println("Both --instance-ids and --config-path flags are required")
				_ = cmd.Help()
				os.Exit(1)
			}

			// Parse the comma-separated instance IDs
			instanceIDSlice := strings.Split(instanceIDs, ",")
			for i, id := range instanceIDSlice {
				instanceIDSlice[i] = strings.TrimSpace(id)
			}

			// Parse the optional attributes to check
			var attrSlice []string
			if attributesToCheck != "" {
				attrSlice = strings.Split(attributesToCheck, ",")
				for i, attr := range attrSlice {
					attrSlice[i] = strings.TrimSpace(attr)
				}
			}

			// Create orchestrator config
			config := orchestrator.Config{
				InstanceIDs:       instanceIDSlice,
				ConfigPath:        configPath,
				AttributesToCheck: attrSlice,
				OutputFormat:      outputFormat,
				ConcurrencyLimit:  concurrencyLimit,
			}

			// Create orchestrator service
			service, err := orchestrator.NewDefaultService(config)
			if err != nil {
				log.Fatalf("Failed to initialize the service: %v", err)
			}

			ctx := context.Background()
			hasDrift, hasError, err := service.Run(ctx)

			if err != nil {
				log.Fatalf("Error: %v", err)
			}

			// Set exit code based on whether drift was detected
			if hasDrift {
				os.Exit(2) // Non-zero exit code indicates drift detected
			}
			if hasError {
				os.Exit(1) // Error occurred during execution
			}
		},
	}

	// Define flags
	rootCmd.Flags().StringVar(&instanceIDs, "instance-ids", "", "Comma-separated list of AWS EC2 instance IDs")
	rootCmd.Flags().StringVar(&configPath, "config-path", "", "Path to the Terraform configuration file")
	rootCmd.Flags().StringVar(&attributesToCheck, "attributes", "", "Comma-separated list of attributes to check for drift (e.g., instance_type,tags)")
	rootCmd.Flags().StringVar(&outputFormat, "output", "table", "Output format: table or json")
	rootCmd.Flags().IntVar(&concurrencyLimit, "concurrency", runtime.NumCPU(), "Maximum number of instances to check concurrently (default: number of CPU cores)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
