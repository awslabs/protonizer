package cmd

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/hack-pad/hackpadfs"
	hackpados "github.com/hack-pad/hackpadfs/os"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/jritsema/scaffolder"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	protonInfrastructureDirEnv = "infrastructure"
	protonInfrastructureDirSvc = "instance_infrastructure"
	protonPipelineDirSvc       = "pipeline_infrastructure"
	protonTFSrc                = "src"
	provisioningTypeCodeBuild  = "codebuild"
	toolTerraform              = "terraform"
	provisioningTypeAWSManaged = "awsmanaged"
)

var (

	//cli flags
	flagProtonizeName                       string
	flagProtonizeSrcDir                     string
	flagProtonizeOutDir                     string
	flagProtonizeProvisoning                string
	flagProtonizeTool                       string
	flagProtonizeTemplateType               string
	flagProtonizePublish                    bool
	flagProtonizeTerraformRemoteStateBucket string
	flagProtonizePublishBucket              string
	flagProtonizeCompatibleEnvs             []string

	tfEnvInfraSrcDir string
	tfSvcInfraSrcDir string
)

// upCmd represents the up command
var templateProtonizeCmd = &cobra.Command{
	Use:   "protonize",
	Short: "Protonize converts existing IaC to Proton",
	Long: `Protonize converts existing IaC to Proton's format so that it can be published.
Currently only supports Terraform using CodeBuild provisioning.`,
	Run: doTemplateProtonize,
	Example: `
# Convert existing Terraform into a Proton environment template
protonizer protonize \
  --name my_template \
  --type environment \
  --dir ~/my-existing-tf-module

# Convert existing Terraform into a Proton service template and publish it
protonizer protonize \
  --name my_template \
  --type service \
  --compatible-env env1:1 --compatible-env env2:1 \
  --provisioning codebuild --tool terraform \
  --dir ~/my-existing-tf-module \
  --bucket my-s3-bucket \
  --publish`,
}

type generateInput struct {
	name                       string
	templateType               string
	srcDir                     string
	srcFS                      hackpadfs.FS
	destFS                     hackpadfs.FS
	publishBucket              string
	terraformRemoteStateBucket string
	compatibleEnvironments     []string
}

type schemaVariable struct {
	Name        string
	Title       string
	Type        string
	Description string
	Default     interface{}
	Required    bool
	ArrayType   string
}

type outputData struct {
	ModuleName string
	Outputs    []tfconfig.Output
}

type terraformManifest struct {
	TemplateName           string
	TemplateType           string
	TerraformS3StateBucket string
}

type terraformMain struct {
	ModuleName string
	Variables  []schemaVariable
}

func init() {
	templateProtonizeCmd.Flags().StringVarP(&flagProtonizeName, "name", "n", "", "The name of the template")
	templateProtonizeCmd.MarkFlagRequired("name")

	templateProtonizeCmd.Flags().StringVarP(&flagProtonizeTemplateType, "type", "t", "environment",
		"Template type: environment or service")

	templateProtonizeCmd.Flags().StringVarP(&flagProtonizeSrcDir, "dir", "s", "",
		"The source directory of the template to parse")
	templateProtonizeCmd.MarkFlagRequired("dir")

	templateProtonizeCmd.Flags().StringVarP(&flagProtonizeOutDir, "out", "o", ".",
		"The directory to output the protonized template. Defaults to the current directory")

	templateProtonizeCmd.Flags().StringVarP(&flagProtonizeProvisoning, "provisioning", "p",
		provisioningTypeCodeBuild, "The provisioning mode to use")

	templateProtonizeCmd.Flags().StringVar(&flagProtonizeTool, "tool", toolTerraform,
		"The tool to use. Currently, only Terraform is supported")

	templateProtonizeCmd.Flags().BoolVar(&flagProtonizePublish, "publish", false,
		"Whether or not to publish the protonized template")

	templateProtonizeCmd.Flags().StringVarP(&flagProtonizePublishBucket, "publish-bucket", "b", "",
		"The S3 bucket to use for template publishing. This is optional if not using the publish command.")

	templateProtonizeCmd.Flags().StringVar(&flagProtonizeTerraformRemoteStateBucket, "terraform-remote-state-bucket", "",
		"The S3 bucket to use for storing Terraform remote state. This is required for --provisioning codebuild and --tool terraform")

	templateProtonizeCmd.Flags().StringArrayVar(&flagProtonizeCompatibleEnvs, "compatible-env", []string{},
		`Proton environments (name:majorversion) that the service template is compatible with.
You may specify any number of environments by repeating --compatible-env before each one`)

	rootCmd.AddCommand(templateProtonizeCmd)

	//env and svc specific TF src directories
	tfEnvInfraSrcDir = path.Join(protonInfrastructureDirEnv, protonTFSrc)
	tfSvcInfraSrcDir = path.Join(protonInfrastructureDirSvc, protonTFSrc)
}

func doTemplateProtonize(cmd *cobra.Command, args []string) {

	//check required args

	if !(flagProtonizeTemplateType == "environment" || flagProtonizeTemplateType == "service") {
		errorExit(fmt.Sprintf("template type: %s is invalid. only environment and service are supported",
			flagProtonizeTemplateType))
	}

	if flagProtonizeTemplateType == "service" && len(flagProtonizeCompatibleEnvs) == 0 {
		errorExit("--compatible-env is required for service templates")
	}

	if flagProtonizeProvisoning != provisioningTypeCodeBuild {
		errorExit("currently the only provisioning type supported is", provisioningTypeCodeBuild)
	}

	if flagProtonizeTool != toolTerraform {
		errorExit("currently the only provisioning type supported is", toolTerraform)
	}

	if flagProtonizeProvisoning == "CodeBuild" && flagProtonizeTool == toolTerraform &&
		flagProtonizeTerraformRemoteStateBucket == "" {
		errorExit("--terraform-remote-state-bucket is required for --provisioning codebuild and --tool terraform")
	}

	//create an os file system rooted at output path
	//the scaffold function will write to this file system
	osfs := hackpados.NewFS()

	//we will output to this file system
	out, err := filepath.Abs(flagProtonizeOutDir)
	handleError("getting absolute path of out dir", err)
	fsPath, err := osfs.FromOSPath(out)
	handleError("FromOSPath", err)
	m := "creating out file system: " + fsPath
	debug(m)
	outFS, err := osfs.Sub(fsPath)
	handleError(m, err)

	//we will copy the user's terraform source into this file system
	sDir, err := filepath.Abs(flagProtonizeSrcDir)
	handleError("getting absolute path of src dir", err)
	fsPath, err = osfs.FromOSPath(sDir)
	handleError("FromOSPath", err)
	m = "creating src file system: " + fsPath
	debug(m)
	srcFS, err := osfs.Sub(fsPath)
	handleError(m, err)

	//generate proton template
	input := generateInput{
		name:                       flagProtonizeName,
		templateType:               flagProtonizeTemplateType,
		srcDir:                     sDir,
		srcFS:                      srcFS,
		destFS:                     outFS,
		publishBucket:              flagProtonizePublishBucket,
		terraformRemoteStateBucket: flagProtonizeTerraformRemoteStateBucket,
		compatibleEnvironments:     flagProtonizeCompatibleEnvs,
	}
	err = generateCodeBuildTerraformTemplate(input)
	handleError("generating template", err)

	templateDir := path.Join(out, flagProtonizeName)
	fmt.Println("template source outputted to", templateDir)

	if flagProtonizePublish {
		publishTemplate(path.Join(templateDir, "v1", "proton.yaml"))
	}

	fmt.Println("done")
}

// generates a proton template and returns the outputted template directory
func generateCodeBuildTerraformTemplate(in generateInput) error {
	debug("name =", in.name)

	//create datasets that gets fed into templates

	//parse input/output variables
	vars, outputs := parseTerraformSource(in.name, in.srcDir)

	mainData := terraformMain{
		ModuleName: in.name,
		Variables:  vars,
	}

	manifestData := terraformManifest{
		TemplateName:           in.name,
		TerraformS3StateBucket: in.terraformRemoteStateBucket,
		TemplateType:           string(in.templateType),
	}

	//codegen proton config
	protonData := protonConfigData{
		Name:                   in.name,
		Type:                   string(in.templateType),
		DisplayName:            in.name,
		Description:            fmt.Sprintf("A %s template generated from %s", in.templateType, in.name),
		PublishBucket:          flagProtonizePublishBucket,
		CompatibleEnvironments: in.compatibleEnvironments,
	}
	protonConfig, err := yaml.Marshal(protonData)
	handleError("marshalling proton config yaml", err)

	tType := getTemplateTypeShorthand(in.templateType)
	root := path.Join(in.name, "v1")
	infraDir := path.Join(root, getInfrastructureDirectory(string(in.templateType)))

	contents := scaffolder.FSContents{
		path.Join(root, "README.md"):                readTemplateFS("readme/%s.tf.md", tType),
		path.Join(root, "proton.yaml"):              protonConfig,
		path.Join(root, "schema/schema.yaml"):       render("schema/schema.%s.yaml.go.tpl", vars, tType),
		path.Join(infraDir, "manifest.yaml"):        render("infrastructure/codebuild/terraform/manifest.yaml.go.tpl", manifestData),
		path.Join(infraDir, "main.tf"):              render("infrastructure/codebuild/terraform/main.%s.tf.go.tpl", mainData, tType),
		path.Join(infraDir, "outputs.tf"):           render("infrastructure/codebuild/terraform/outputs.tf.go.tpl", outputs),
		path.Join(infraDir, "output.sh"):            readTemplateFS("infrastructure/codebuild/terraform/output.sh"),
		path.Join(infraDir, "variables.tf"):         readTemplateFS("infrastructure/codebuild/terraform/variables.%s.tf", tType),
		path.Join(infraDir, "install-terraform.sh"): readTemplateFS("infrastructure/codebuild/terraform/install-terraform.sh"),
	}

	//populate the file system with the generated contents
	err = scaffolder.PopulateFS(in.destFS, contents)
	if err != nil {
		return err
	}

	//copy terraform src filesystem to infrastructure/src
	outDir := path.Join(infraDir, protonTFSrc)
	destFS, err := hackpadfs.Sub(in.destFS, outDir)
	handleError("creating file system", err)
	m := "copying filesystem"
	debug(m)
	err = scaffolder.CopyFS(in.srcFS, destFS)
	handleError(m, err)

	return nil
}

// returns the name of the infrastructure directory based on the template type
func getInfrastructureDirectory(templateType string) string {
	if templateType == "environment" {
		return protonInfrastructureDirEnv
	}
	return protonInfrastructureDirSvc
}

// returns the template type shorthand (env or svc)
func getTemplateTypeShorthand(templateType string) string {
	if templateType == "environment" {
		return "env"
	}
	return "svc"
}

func parseTerraformSource(name, srcDir string) ([]schemaVariable, outputData) {

	m := "parsing terraform module: " + srcDir
	debug(m)
	module, diags := tfconfig.LoadModule(srcDir)
	if err := diags.Err(); err != nil {
		handleError(m, err)
	}
	debug("\n")
	debugFmt("found %v variables", len(module.Variables))
	debug("\n")

	//sort variables by name
	inputVars := sortTFVariables(module)

	//map tf variables to openapi properties
	vars := []schemaVariable{}
	for _, v := range inputVars {
		debugFmt("%v (type: %v; default: %v) \n", v.Name, v.Type, v.Default)

		//escape quotes in descriptions
		desc := strings.Replace(v.Description, `"`, `\"`, -1)

		sv := schemaVariable{
			Name:        v.Name,
			Title:       v.Name,
			Type:        v.Type,
			Description: desc,
			Default:     v.Default,
			Required:    v.Required,
		}

		//default values
		if v.Default != nil {
			sv.Default = v.Default
		}

		if v.Type == "bool" {
			sv.Type = "boolean"
		}

		//list(x) -> array of x
		if strings.HasPrefix(sv.Type, "list(") {
			sv.Type = "array"
			sv.ArrayType = strings.Split(v.Type, "list(")[1]
			sv.ArrayType = sv.ArrayType[:len(sv.ArrayType)-1]
			sv.Default = nil
		}

		//output warning for unsupported types
		if strings.HasPrefix(sv.Type, "object(") ||
			strings.HasPrefix(sv.Type, "any") ||
			strings.HasPrefix(sv.Type, "map(") ||
			strings.HasPrefix(sv.Type, "set(") {

			fmt.Println("WARNING: skipping unsupported input variable:")
			fmt.Println(v.Name)
			fmt.Println(v.Type)
			fmt.Println()
			continue
		}

		vars = append(vars, sv)
	}

	//debug
	if verbose {
		debug("\n")
		debugFmt("found %v outputs", len(module.Outputs))
		debug("\n")
		for _, o := range module.Outputs {
			debugFmt("%v (description: %v) \n", o.Name, o.Description)
		}
	}

	//return output
	outputs := outputData{ModuleName: name}
	outputs.Outputs = sortTFOutputs(module)

	return vars, outputs
}
