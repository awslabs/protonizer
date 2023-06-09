package cmd

import (
	"errors"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/jritsema/scaffolder"
	"gopkg.in/yaml.v3"
)

func TestNewEnvironmentTemplateCloudFormation(t *testing.T) {

	//create in-memory file system for testing
	destFS, err := mem.NewFS()
	if err != nil {
		t.Error(err)
	}

	name := "my-template"

	scaffoldProton(
		name,
		"environment",
		"awsmanaged",
		"", //tool
		"my-s3-bucket",
		"my-s3-bucket",
		[]string{},
		destFS,
	)

	//prepend template name to output directories
	schemaDir := path.Join(name, "schema")
	infraDir := path.Join(name, protonInfrastructureDirEnv)

	pathsToCheck := []string{
		path.Join(name, "proton.yaml"),
		path.Join(schemaDir, "schema.yaml"),
		path.Join(infraDir, "manifest.yaml"),
		path.Join(infraDir, "cloudformation.yaml"),
	}

	internalCheckPaths(t, destFS, pathsToCheck)
}

func TestNewServiceTemplateCloudFormation(t *testing.T) {

	//create in-memory file system for testing
	destFS, err := mem.NewFS()
	if err != nil {
		t.Error(err)
	}

	name := "my-template"

	scaffoldProton(
		name,
		"service",
		"awsmanaged",
		"", //tool
		"my-s3-bucket",
		"my-s3-bucket",
		[]string{"my-template:1"},
		destFS,
	)

	//prepend template name to output directories
	schemaDir := path.Join(name, "schema")
	infraDir := path.Join(name, protonInfrastructureDirSvc)

	pathsToCheck := []string{
		path.Join(name, "proton.yaml"),
		path.Join(schemaDir, "schema.yaml"),
		path.Join(infraDir, "manifest.yaml"),
		path.Join(infraDir, "cloudformation.yaml"),
	}

	internalCheckPaths(t, destFS, pathsToCheck)
}

func TestNewEnvironmentTemplateCodeBuildTerraform(t *testing.T) {

	//create in-memory file system for testing
	destFS, err := mem.NewFS()
	if err != nil {
		t.Error(err)
	}

	name := "my-template"

	scaffoldProton(
		name,
		"environment",
		"codebuild",
		"terraform",
		"my-publish-bucket",
		"my-remote-state-bucket",
		[]string{},
		destFS,
	)

	//prepend template name to output directories
	schemaDir := path.Join(name, "schema")
	infraDir := path.Join(name, protonInfrastructureDirEnv)

	pathsToCheck := []string{
		path.Join(name, "proton.yaml"),
		path.Join(schemaDir, "schema.yaml"),
		path.Join(infraDir, "manifest.yaml"),
		path.Join(infraDir, "main.tf"),
		path.Join(infraDir, "variables.tf"),
		path.Join(infraDir, "outputs.tf"),
		path.Join(infraDir, "output.sh"),
		path.Join(infraDir, "install-terraform.sh"),
	}

	internalCheckPaths(t, destFS, pathsToCheck)
}

func TestNewServiceTemplateCodeBuildTerraform(t *testing.T) {

	//create in-memory file system for testing
	destFS, err := mem.NewFS()
	if err != nil {
		t.Error(err)
	}

	name := "my-template"

	scaffoldProton(
		name,
		"service",
		"codebuild",
		"terraform",
		"my-publish-bucket",
		"my-remote-state-bucket",
		[]string{"my-env:1"},
		destFS,
	)

	//prepend template name to output directories
	schemaDir := path.Join(name, "schema")
	infraDir := path.Join(name, protonInfrastructureDirSvc)

	pathsToCheck := []string{
		path.Join(name, "proton.yaml"),
		path.Join(schemaDir, "schema.yaml"),
		path.Join(infraDir, "manifest.yaml"),
		path.Join(infraDir, "main.tf"),
		path.Join(infraDir, "variables.tf"),
		path.Join(infraDir, "outputs.tf"),
		path.Join(infraDir, "output.sh"),
		path.Join(infraDir, "install-terraform.sh"),
	}

	internalCheckPaths(t, destFS, pathsToCheck)
}

func internalCheckPaths(t *testing.T, destFS fs.FS, pathsToCheck []string) {
	scaffolder.InspectFS(destFS, t.Log, false)

	t.Log("pathsToCheck")
	t.Log(pathsToCheck)

	findings := 0
	err := hackpadfs.WalkDir(destFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		t.Log("looking for path in destFS", path)
		if SliceContains(&pathsToCheck, path, false) {
			findings++
			t.Log("found", path)
		}

		//test that any generated yaml is valid
		if (strings.HasSuffix(path, "yaml") || strings.HasSuffix(path, "yml")) &&
			!strings.HasSuffix(path, "cloudformation.yaml") {
			t.Log("testing", path)
			contents, err := hackpadfs.ReadFile(destFS, path)
			t.Log(string(contents))
			if err != nil {
				t.Error(err)
			}
			var data interface{}
			err = yaml.Unmarshal(contents, &data)
			if err != nil {
				t.Error("invalid generated yaml", err)
			}
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
	t.Log("expected paths:", len(pathsToCheck))
	t.Log("actual paths:", findings)

	if findings != len(pathsToCheck) {
		t.Error(errors.New("path counts don't match. did you add/remove something in the local templates directory?"))
	}
}
