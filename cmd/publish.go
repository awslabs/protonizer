package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/proton"
	"github.com/aws/aws-sdk-go-v2/service/proton/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	flagTemplatePublishFile string
)

type protonConfigData struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	DisplayName string `yaml:"displayName"`
	Description string `yaml:"description"`

	//optional
	PublishBucket          string   `yaml:"publishBucket,omitempty"`
	CompatibleEnvironments []string `yaml:"compatibleEnvironments,omitempty"`
}

var templatePublishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publishes proton templates",
	Long:  "Publishes proton templates",
	Run:   doTemplatePublish,
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	templatePublishCmd.Flags().StringVarP(&flagTemplatePublishFile, "file", "f", "proton.yaml", "The proton yaml file to use")
	rootCmd.AddCommand(templatePublishCmd)
}

func getAWSConfig() aws.Config {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	handleError("aws config", err)
	return cfg
}

func doTemplatePublish(cmd *cobra.Command, args []string) {
	publishTemplate(flagTemplatePublishFile)
}

func publishTemplate(file string) {

	//parse proton.yaml
	protonConfig, err := readProtonYAMLFile(file)
	if err != nil {
		errorExit(fmt.Errorf("could not read proton.yaml: %w", err))
	}

	if protonConfig.PublishBucket == "" {
		errorExit("The `publishBucket` key is not specified in proton.yaml.  This setting is required for publishing.")
	}

	//tar gz template
	//assume template bundle is in the same directory as the proton.yaml file
	dir := filepath.Dir(file)
	zipFileName := "bundle.tar.gz"
	zipPath := path.Join(dir, zipFileName)
	m := "creating template bundle: " + zipPath
	debug(m)
	err = createTarGZFile(dir, zipPath)
	handleError(m, err)

	cfg := getAWSConfig()
	ctx := context.Background()

	//upload to s3
	bucket := protonConfig.PublishBucket
	key := zipFileName

	s3Client := s3.NewFromConfig(cfg)

	m = fmt.Sprintf("uploading template bundle to s3://%s/%s", bucket, key)
	debug(m)
	f, err := os.Open(zipPath)
	handleError("opening zip", err)
	uploader := manager.NewUploader(s3Client)
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: &bucket,
		Key:    &key,
		Body:   f,
	})
	handleError(m, err)

	//delete local zip file
	err = os.Remove(zipPath)
	handleError("removing zip file", err)

	//publish
	var majorVersion, minorVersion string

	switch protonConfig.Type {

	case "environment":
		majorVersion, minorVersion = publishEnvironmentTemplate(cfg, protonConfig, key, ctx)

	case "service":
		majorVersion, minorVersion = publishServiceTemplate(cfg, protonConfig, key, ctx)
	}
	fmt.Printf("published %s:%s.%s \n", protonConfig.Name, majorVersion, minorVersion)

	//output console url of published template
	fmt.Printf("https://%s.console.aws.amazon.com/proton/home#/templates/%vs/detail/%s\n",
		cfg.Region, protonConfig.Type, protonConfig.Name)
}

func publishEnvironmentTemplate(cfg aws.Config, protonConfig *protonConfigData, s3Key string, ctx context.Context) (string, string) {

	//publish proton template and version
	protonClient := proton.NewFromConfig(cfg)

	reqTemplate := &proton.CreateEnvironmentTemplateInput{
		Name:        &protonConfig.Name,
		Description: &protonConfig.Description,
		DisplayName: &protonConfig.DisplayName,
		Tags: []types.Tag{
			{
				Key:   aws.String("creator"),
				Value: aws.String("protonizer-cli"),
			},
		},
	}
	m := "proton.CreateEnvironmentTemplate()"
	debug(m)
	_, err := protonClient.CreateEnvironmentTemplate(ctx, reqTemplate)
	handleError(m, err)

	//publish version
	majorVesion := "1"

	s3Source := types.TemplateVersionSourceInputMemberS3{
		Value: types.S3ObjectSource{
			Bucket: &protonConfig.PublishBucket,
			Key:    &s3Key,
		},
	}
	reqVersion := &proton.CreateEnvironmentTemplateVersionInput{
		TemplateName: &protonConfig.Name,
		MajorVersion: &majorVesion,
		Source:       &s3Source,
	}
	m = "proton.CreateEnvironmentTemplateVersion()"
	debug(m)
	templateVersion, err := protonClient.CreateEnvironmentTemplateVersion(ctx, reqVersion)
	handleError(m, err)

	if templateVersion.EnvironmentTemplateVersion.MinorVersion != nil {
		debug("minor version =", *templateVersion.EnvironmentTemplateVersion.MinorVersion)
	}
	debug(templateVersion.EnvironmentTemplateVersion.Status)
	if templateVersion.EnvironmentTemplateVersion.StatusMessage != nil {
		debug(*templateVersion.EnvironmentTemplateVersion.StatusMessage)
	}

	debug("waiting for registration to complete")

	//wait for version to be available then get the minor version
	minorVersion := ""
	for {
		m = "proton.GetEnvironmentTemplate()"
		debug(m)
		ver, err := protonClient.GetEnvironmentTemplateVersion(ctx, &proton.GetEnvironmentTemplateVersionInput{
			TemplateName: &protonConfig.Name,
			MajorVersion: &majorVesion,
			MinorVersion: templateVersion.EnvironmentTemplateVersion.MinorVersion,
		})
		handleError(m, err)
		debug(ver.EnvironmentTemplateVersion.Status)
		if ver.EnvironmentTemplateVersion.StatusMessage != nil {
			debug(*ver.EnvironmentTemplateVersion.StatusMessage)
		}

		if ver.EnvironmentTemplateVersion.Status == types.TemplateVersionStatusRegistrationFailed {
			errorExit(*ver.EnvironmentTemplateVersion.StatusMessage)
		}

		if ver.EnvironmentTemplateVersion.Status == types.TemplateVersionStatusDraft {
			minorVersion = *ver.EnvironmentTemplateVersion.MinorVersion
			debugFmt("template version %s.%s now in %s", majorVesion, minorVersion, ver.EnvironmentTemplateVersion.Status)
			break
		}
		time.Sleep(2 * time.Second)
	}

	desc := "published by proton cli"
	m = "proton.UpdateEnvironmentTemplateVersion"
	debug(m)
	_, err = protonClient.UpdateEnvironmentTemplateVersion(ctx, &proton.UpdateEnvironmentTemplateVersionInput{
		TemplateName: &protonConfig.Name,
		MajorVersion: &majorVesion,
		MinorVersion: &minorVersion,
		Status:       types.TemplateVersionStatusPublished,
		Description:  &desc,
	})
	handleError(m, err)

	return majorVesion, minorVersion
}

func publishServiceTemplate(cfg aws.Config, protonConfig *protonConfigData, s3Key string, ctx context.Context) (string, string) {

	//publish proton template and version
	protonClient := proton.NewFromConfig(cfg)

	reqTemplate := &proton.CreateServiceTemplateInput{
		Name:                 &protonConfig.Name,
		Description:          &protonConfig.Description,
		DisplayName:          &protonConfig.DisplayName,
		PipelineProvisioning: types.ProvisioningCustomerManaged,
		Tags: []types.Tag{
			{
				Key:   aws.String("creator"),
				Value: aws.String("protonizer-cli"),
			},
		},
	}
	m := "proton.CreateServiceTemplate()"
	debug(m)
	_, err := protonClient.CreateServiceTemplate(ctx, reqTemplate)
	handleError(m, err)

	//publish version
	majorVesion := "1"

	s3Source := types.TemplateVersionSourceInputMemberS3{
		Value: types.S3ObjectSource{
			Bucket: &protonConfig.PublishBucket,
			Key:    &s3Key,
		},
	}
	reqVersion := &proton.CreateServiceTemplateVersionInput{
		TemplateName:                   &protonConfig.Name,
		MajorVersion:                   &majorVesion,
		Source:                         &s3Source,
		CompatibleEnvironmentTemplates: []types.CompatibleEnvironmentTemplateInput{},
	}

	for _, c := range protonConfig.CompatibleEnvironments {
		parts := strings.Split(c, ":")
		if len(parts) != 2 {
			errorExit("compatible environments must use the format: `name:version`")
		}
		reqVersion.CompatibleEnvironmentTemplates = append(reqVersion.CompatibleEnvironmentTemplates, types.CompatibleEnvironmentTemplateInput{
			TemplateName: aws.String(parts[0]),
			MajorVersion: aws.String(parts[1]),
		})
	}

	m = "proton.CreateServiceTemplateVersion()"
	debug(m)
	templateVersion, err := protonClient.CreateServiceTemplateVersion(ctx, reqVersion)
	if err != nil {
		if strings.Contains(err.Error(), "ValidationException") {
			errorExit(err)
		} else {
			handleError(m, err)
		}
	}

	if templateVersion.ServiceTemplateVersion.MinorVersion != nil {
		debug("minor version =", *templateVersion.ServiceTemplateVersion.MinorVersion)
	}
	debug(templateVersion.ServiceTemplateVersion.Status)
	if templateVersion.ServiceTemplateVersion.StatusMessage != nil {
		debug(*templateVersion.ServiceTemplateVersion.StatusMessage)
	}

	debug("waiting for registration to complete")

	//wait for version to be available then get the minor version
	minorVersion := ""
	for {
		m = "proton.GetServiceTemplate()"
		debug(m)
		ver, err := protonClient.GetServiceTemplateVersion(ctx, &proton.GetServiceTemplateVersionInput{
			TemplateName: &protonConfig.Name,
			MajorVersion: &majorVesion,
			MinorVersion: templateVersion.ServiceTemplateVersion.MinorVersion,
		})
		handleError(m, err)
		debug(ver.ServiceTemplateVersion.Status)
		if ver.ServiceTemplateVersion.StatusMessage != nil {
			debug(*ver.ServiceTemplateVersion.StatusMessage)
		}

		if ver.ServiceTemplateVersion.Status == types.TemplateVersionStatusRegistrationFailed {
			errorExit(*ver.ServiceTemplateVersion.StatusMessage)
		}

		if ver.ServiceTemplateVersion.Status == types.TemplateVersionStatusDraft {
			minorVersion = *ver.ServiceTemplateVersion.MinorVersion
			debugFmt("template version %s.%s now in %s", majorVesion, minorVersion, ver.ServiceTemplateVersion.Status)
			break
		}
		time.Sleep(2 * time.Second)
	}

	desc := "published by proton cli"
	m = "proton.UpdateEnvironmentTemplateVersion"
	debug(m)
	_, err = protonClient.UpdateServiceTemplateVersion(ctx, &proton.UpdateServiceTemplateVersionInput{
		TemplateName: &protonConfig.Name,
		MajorVersion: &majorVesion,
		MinorVersion: &minorVersion,
		Status:       types.TemplateVersionStatusPublished,
		Description:  &desc,
	})
	handleError(m, err)

	return majorVesion, minorVersion
}

func readProtonYAMLFile(fileName string) (*protonConfigData, error) {

	yamlFile, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %s: %w", fileName, err)
	}
	defer yamlFile.Close()
	b, err := io.ReadAll(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %s : %w", fileName, err)
	}
	var result protonConfigData
	err = yaml.Unmarshal(b, &result)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling file: %s : %w", fileName, err)
	}
	return &result, nil
}
