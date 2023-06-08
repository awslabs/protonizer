/*
Copyright © 2022 John Ritsema
*/
package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

// strongly-typed template type (environment or service)
type protonTemplateType string

const (
	templateTypeEnvironment protonTemplateType = "environment"
	templateTypeService     protonTemplateType = "service"
)

// verbose logging enabled
var verbose = false

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "protonizer",
	Short: "A CLI tool for working with IaC in AWS Proton.",
	Long:  "A CLI tool for working with IaC in AWS Proton.",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(version string) {
	rootCmd.Version = version
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
}
