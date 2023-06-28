package cmd

import (
	"embed"
	"log"
	"os"
	"text/template"

	"github.com/spf13/cobra"
)

//go:embed templates/*
var templateFS embed.FS

// verbose logging enabled
var verbose = false

// parsed scaffold templates
var scaffoldTemplates *template.Template

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

	//parse go templates
	debug("parsing go templates")
	var err error
	scaffoldTemplates, err = templateParseFSRecursive(templateFS, ".tpl", nil)
	handleError("error parsing go templates", err)
	debugFmt("defined templates: %v", scaffoldTemplates.DefinedTemplates())
}
