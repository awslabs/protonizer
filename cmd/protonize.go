package cmd

import (
	"embed"
	_ "embed"
	"fmt"
	"path"
	"strings"
	"text/template"

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
	protonSchemaDir            = "schema"
	protonTFSrc                = "src"
	toolTerraform              = "terraform"
	provisioningTypeCodeBuild  = "codebuild"
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
	flagProtonizeCompatibleEnvs             []string

	tfEnvInfraSrcDir string
	tfSvcInfraSrcDir string
)

//go:embed templates/*
var templateFS embed.FS

// parsed scaffold templates
var scaffoldTemplates *template.Template

// upCmd represents the up command
var templateProtonizeCmd = &cobra.Command{
	Use:   "protonize",
	Short: "Protonize converts existing IaC to Proton",
	Long: `Protonize converts existing IaC to Proton's format so that it can be published.
Currently only supports Terraform using CodeBuild provisioning.`,
	Run: doTemplateProtonize,
	Example: `protonizer protonize \
  --name my_template \
  --type service \
  --compatible-env env1:1 --compatible-env env2:1 \
  --tool terraform \
  --provisioning codebuild \
  --dir ~/my-existing-tf-module \
  --out ~/proton/templates \
  --bucket my-s3-bucket \
  --publish`,
}

type generateInput struct {
	name                       string
	templateType               protonTemplateType
	srcDir                     string
	srcFS                      hackpadfs.FS
	destFS                     hackpadfs.FS
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
	TerraformS3StateBucket string
	TemplateType           string
}

type terraformMain struct {
	ModuleName string
	Variables  []schemaVariable
}

func init() {

	var err error
	scaffoldTemplates, err = templateParseFSRecursive(templateFS, ".tpl", nil)
	handleError("error parsing go templates", err)

	templateProtonizeCmd.Flags().StringVarP(&flagProtonizeName, "name", "n", "", "The name of the template")
	templateProtonizeCmd.MarkFlagRequired("name")

	templateProtonizeCmd.Flags().StringVarP(&flagProtonizeTemplateType, "type", "t", "environment", "Template type: environment or service")

	templateProtonizeCmd.Flags().StringVarP(&flagProtonizeSrcDir, "dir", "s", "", "The source directory of the template to parse")
	templateProtonizeCmd.MarkFlagRequired("dir")

	templateProtonizeCmd.Flags().StringVarP(&flagProtonizeOutDir, "out", "o", ".", "The directory to output the protonized template")

	templateProtonizeCmd.Flags().StringVarP(&flagProtonizeProvisoning, "provisioning", "p", provisioningTypeCodeBuild, "The provisioning mode to use")

	templateProtonizeCmd.Flags().StringVar(&flagProtonizeTool, "tool", toolTerraform, "The tool to use. Currently, only Terraform is supported")

	templateProtonizeCmd.Flags().BoolVar(&flagProtonizePublish, "publish", false, "Whether or not to publish the protonized template")

	templateProtonizeCmd.Flags().StringVarP(&flagProtonizeTerraformRemoteStateBucket, "bucket", "b", "", "The S3 bucket to use for storing Terraform remote state")
	templateProtonizeCmd.MarkFlagRequired("bucket")

	templateProtonizeCmd.Flags().StringArrayVar(&flagProtonizeCompatibleEnvs, "compatible-env", []string{},
		"Proton environments (name:majorversion) that the service template is compatible with. You may specify any number of environments by repeating --compatible-env before each one")

	rootCmd.AddCommand(templateProtonizeCmd)

	//env and svc specific TF src directories
	tfEnvInfraSrcDir = path.Join(protonInfrastructureDirEnv, protonTFSrc)
	tfSvcInfraSrcDir = path.Join(protonInfrastructureDirSvc, protonTFSrc)
}

func doTemplateProtonize(cmd *cobra.Command, args []string) {

	//check required args
	if flagProtonizeTool != toolTerraform {
		errorExit("currently the only provisioning type supported is", toolTerraform)
	}
	if flagProtonizeProvisoning != provisioningTypeCodeBuild {
		errorExit("currently the only provisioning type supported is", provisioningTypeCodeBuild)
	}

	//create a file system rooted at output path (remove trailing "/")
	//the generate function will write to this file system
	osfs := hackpados.NewFS()
	srcFS, err := osfs.Sub(flagProtonizeSrcDir[1:])
	handleError("creating src file system", err)

	outFS, err := osfs.Sub(flagProtonizeOutDir[1:])
	handleError("creating out file system", err)

	var tType protonTemplateType
	switch flagProtonizeTemplateType {
	case string(templateTypeEnvironment):
		tType = templateTypeEnvironment
	case string(templateTypeService):
		tType = templateTypeService
	default:
		errorExit(fmt.Sprintf("template type: %s is invalid. only %v and %v are supported",
			flagProtonizeTemplateType, templateTypeEnvironment, templateTypeService))
	}

	//generate proton template
	input := generateInput{
		name:                       flagProtonizeName,
		templateType:               tType,
		srcDir:                     flagProtonizeSrcDir,
		srcFS:                      srcFS,
		destFS:                     outFS,
		terraformRemoteStateBucket: flagProtonizeTerraformRemoteStateBucket,
		compatibleEnvironments:     flagProtonizeCompatibleEnvs,
	}
	err = generateCodeBuildTerraformTemplate(input)
	handleError("generating template", err)

	templateDir := path.Join(flagProtonizeOutDir, flagProtonizeName)
	fmt.Println("template source outputted to", templateDir)

	if flagProtonizePublish {
		publishTemplate(path.Join(templateDir, ProtonYamlFile))
	}

	fmt.Println("done")
}

// generates a proton template and returns the outputted template directory
func generateCodeBuildTerraformTemplate(in generateInput) error {
	debug("name =", in.name)
	debug("srcDir =", in.srcDir)

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
		Name:                       in.name,
		Type:                       string(in.templateType),
		DisplayName:                in.name,
		Description:                fmt.Sprintf("A %s template generated from %s", in.templateType, in.name),
		TerraformRemoteStateBucket: in.terraformRemoteStateBucket,
		CompatibleEnvironments:     in.compatibleEnvironments,
	}
	protonConfig, err := yaml.Marshal(protonData)
	handleError("marshalling proton config yaml", err)

	tType := ""
	infraDir := ""
	switch in.templateType {
	case templateTypeEnvironment:
		tType = "env"
		infraDir = path.Join(in.name, protonInfrastructureDirEnv)
	case templateTypeService:
		tType = "svc"
		infraDir = path.Join(in.name, protonInfrastructureDirSvc)
	}

	schemaDir := path.Join(in.name, protonSchemaDir)

	contents := scaffolder.FSContents{
		path.Join(in.name, ProtonYamlFile):          protonConfig,
		path.Join(schemaDir, "schema.yaml"):         render("schema/schema.%s.yaml.go.tpl", vars, tType),
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
	outDir := path.Join(path.Join(infraDir, protonTFSrc))
	destFS, err := hackpadfs.Sub(in.destFS, outDir)
	handleError("creating file system", err)

	m := "copying filesystem"
	debug(m)
	err = scaffolder.CopyFS(in.srcFS, destFS)
	handleError(m, err)

	return nil
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
