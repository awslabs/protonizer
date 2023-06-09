package cmd

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"

	hackpados "github.com/hack-pad/hackpadfs/os"
	"github.com/jritsema/scaffolder"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// upCmd represents the up command
var newCmd = &cobra.Command{
	Use:   "new",
	Short: "new scaffolds out a new proton template",
	Long:  `new scaffolds out a new proton template`,
	Run:   doNew,
	Example: `
# Create a new environment template using AWS-Managed CloudFormation
protonizer new --name my-env-template --provisioning awsmanaged

# Create a new service template using AWS-Managed CloudFormation
protonizer new --name my-template \
  --provisioning awsmanaged \
  --type service \
  --compatible-env my-env-template:1

# Create a new environment template using CodeBuild provisioning with Terraform
protonizer new \
  --name my_template \
  --provisioning codebuild --tool terraform \
  --terraform-remote-state-bucket my-s3-bucket

# Create a new service template using CodeBuild provisioning with Terraform
protonizer new \
  --name my_template \
  --type service \
  --provisioning codebuild --tool terraform \
  --terraform-remote-state-bucket my-s3-bucket \
  --publish-bucket my-s3-bucket \
  --compatible-env my-env-template:1 \
  --out ~/proton/templates

# If you would like to use protonizer to publish this template,
then you can include an S3 bucket that you have write access to
protonizer new --name my-template --publish-bucket my-s3-bucket
`,
}

var (
	flagNewTemplateName               string
	flagNewTemplateType               string
	flagNewOutDir                     string
	flagNewProvisoning                string
	flagNewTool                       string
	flagNewPublishBucket              string
	flagNewTerraformRemoteStateBucket string
	flagNewCompatibleEnvs             []string
)

func init() {
	newCmd.Flags().StringVarP(&flagNewTemplateName, "name", "n", "", "The name of the template")
	newCmd.MarkFlagRequired("name")

	newCmd.Flags().StringVarP(&flagNewTemplateType, "type", "t", "environment", "Template type: environment or service")

	newCmd.Flags().StringVarP(&flagNewOutDir, "out", "o", ".", "The directory to output the protonized template. Defaults to current directory.")

	newCmd.Flags().StringVarP(&flagNewProvisoning, "provisioning", "p", provisioningTypeCodeBuild, "The provisioning mode to use")

	newCmd.Flags().StringVar(&flagNewTool, "tool", toolTerraform, "The tool to use. Currently Terraform is supported")

	newCmd.Flags().StringVarP(&flagNewPublishBucket, "publish-bucket", "b", "",
		"The S3 bucket to use for template publishing. This is optional if not using the publish command.")

	newCmd.Flags().StringVar(&flagNewTerraformRemoteStateBucket, "terraform-remote-state-bucket", "",
		"The S3 bucket to use for storing Terraform remote state. This is required for --provisioning codebuild and --tool terraform")

	newCmd.Flags().StringArrayVar(&flagNewCompatibleEnvs, "compatible-env", []string{},
		`Proton environments (name:majorversion) that the service template is compatible with.
You may specify any number of environments by repeating --compatible-env before each one`)

	rootCmd.AddCommand(newCmd)
}

type scaffoldInputData struct {
	Contents               *scaffolder.FSContents
	Name                   string
	Type                   string
	Shorthand              string
	RootDir                string
	InfraDir               string
	Vars                   []schemaVariable
	TerraformS3StateBucket string
}

func doNew(cmd *cobra.Command, args []string) {

	//check required args

	if !(flagNewTemplateType == "environment" || flagNewTemplateType == "service") {
		errorExit(fmt.Sprintf("template type: %s is invalid. only environment and service are supported",
			flagProtonizeTemplateType))
	}

	if flagNewTool != toolTerraform {
		errorExit("currently the only provisioning type supported is", toolTerraform)
	}

	if flagNewTemplateType == "service" && len(flagNewCompatibleEnvs) == 0 {
		errorExit("--compatible-env is required for service templates")
	}

	//create a file system rooted at output path
	//the scaffold function will write to this file system
	out, err := filepath.Abs(flagNewOutDir)
	handleError("getting absolute path of out dir", err)
	osfs := hackpados.NewFS()
	fsPath, err := osfs.FromOSPath(out)
	m := "creating out file system: " + fsPath
	debug(m)
	handleError("FromOSPath", err)
	outFS, err := osfs.Sub(fsPath)
	handleError(fsPath, err)

	scaffoldProton(
		flagNewTemplateName,
		flagNewTemplateType,
		flagNewProvisoning,
		flagNewTool,
		flagNewPublishBucket,
		flagNewTerraformRemoteStateBucket,
		flagNewCompatibleEnvs,
		outFS,
	)

	fmt.Println("template source outputted to", path.Join(out, flagNewTemplateName))
	fmt.Println("done")
}

func scaffoldProton(
	name,
	templateType,
	provisioning,
	tool,
	s3Bucket,
	terraformRemoteStateBucket string,
	compatibleEnvironments []string,
	outFS fs.FS) {

	m := "generating proton config"
	debug(m)
	protonData := protonConfigData{
		Name:                   name,
		Type:                   templateType,
		DisplayName:            name,
		Description:            fmt.Sprintf("A %s template scaffolded by the Protonizer CLI tool", templateType),
		PublishBucket:          s3Bucket,
		CompatibleEnvironments: compatibleEnvironments,
	}
	protonConfig, err := yaml.Marshal(protonData)
	handleError(m, err)

	//schema variables
	//proton seems to require at least one input variable
	schemaVars := []schemaVariable{
		{
			Name:        "example_input",
			Type:        "string",
			Title:       "Example Input",
			Description: "This is an example string input",
			Default:     "default",
		},
	}

	tType := getTemplateTypeShorthand(templateType)
	root := path.Join(name, "v1")
	infraDir := path.Join(root, getInfrastructureDirectory(templateType))

	//scaffold common files
	contents := scaffolder.FSContents{
		path.Join(root, "proton.yaml"):           protonConfig,
		path.Join(root, "schema", "schema.yaml"): render("schema/schema.%s.yaml.go.tpl", schemaVars, tType),
	}

	//add proton template-specific content
	in := scaffoldInputData{
		Contents:               &contents,
		Name:                   name,
		Type:                   templateType,
		Shorthand:              getTemplateTypeShorthand(templateType),
		RootDir:                root,
		InfraDir:               infraDir,
		Vars:                   schemaVars,
		TerraformS3StateBucket: terraformRemoteStateBucket,
	}

	if provisioning == provisioningTypeAWSManaged {
		addAWSManagedTemplateContent(in)
	}
	if provisioning == provisioningTypeCodeBuild {
		if flagNewTool == toolTerraform {
			addCBPTerraformTemplateContent(in)
		}
	}

	//populate the file system with the generated contents
	m = "writing to file system"
	debug(m)
	err = scaffolder.PopulateFS(outFS, contents)
	handleError(m, err)
}

func addAWSManagedTemplateContent(in scaffoldInputData) {

	addContent(in.Contents, in.RootDir, "README.md",
		"readme/%s.cfn.md", in.Shorthand)

	addContent(in.Contents, in.InfraDir, "manifest.yaml",
		"infrastructure/awsmanaged/manifest.yaml")

	addContent(in.Contents, in.InfraDir, "cloudformation.yaml",
		"infrastructure/awsmanaged/cloudformation.%s.yaml.jinja", in.Shorthand)
}

func addCBPTerraformTemplateContent(in scaffoldInputData) {
	contents := *in.Contents

	manifestData := terraformManifest{
		TemplateType:           in.Type,
		TemplateName:           in.Name,
		TerraformS3StateBucket: in.TerraformS3StateBucket,
	}
	manifest := render("infrastructure/codebuild/terraform/manifest.yaml.go.tpl", manifestData)
	contents[path.Join(in.InfraDir, "manifest.yaml")] = manifest

	mainData := terraformMain{
		ModuleName: in.Name,
		Variables:  in.Vars,
	}
	contents[path.Join(in.InfraDir, "main.tf")] =
		render("infrastructure/codebuild/terraform/main.%s.tf.new.go.tpl",
			mainData.Variables[0], in.Shorthand)

	contents[path.Join(in.InfraDir, "outputs.tf")] =
		render("infrastructure/codebuild/terraform/outputs.tf.go.tpl",
			outputData{ModuleName: in.Name})

	addContent(in.Contents, in.RootDir, "README.md",
		"readme/%s.tf.md", in.Shorthand)

	addContent(in.Contents, in.InfraDir, "variables.tf",
		"infrastructure/codebuild/terraform/variables.%s.tf", in.Shorthand)

	addContent(in.Contents, in.InfraDir, "output.sh",
		"infrastructure/codebuild/terraform/output.sh")

	addContent(in.Contents, in.InfraDir, "install-terraform.sh",
		"infrastructure/codebuild/terraform/install-terraform.sh")
}
