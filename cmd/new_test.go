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

	pathsToCheck := getExpectedOutputFiles(name, "environment", "awsmanaged", "")
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

	pathsToCheck := getExpectedOutputFiles(name, "service", "awsmanaged", "")
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

	pathsToCheck := getExpectedOutputFiles(name, "environment", "codebuild", "terraform")
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

	pathsToCheck := getExpectedOutputFiles(name, "service", "codebuild", "terraform")
	internalCheckPaths(t, destFS, pathsToCheck)
}

func internalCheckPaths(t *testing.T, destFS fs.FS, pathsToCheck []string) {
	scaffolder.InspectFS(destFS, t.Log, false)

	t.Log("pathsToCheck")
	t.Log(pathsToCheck)

	found := 0
	findings := []string{}
	err := hackpadfs.WalkDir(destFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			findings = append(findings, path)
		}
		t.Log("looking for path in destFS", path)
		if SliceContains(&pathsToCheck, path, false) {
			found++
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
	t.Log("actual paths:", len(findings))

	if len(findings) != len(pathsToCheck) {
		t.Log()
		//show expected files that were not found
		for _, f := range pathsToCheck {
			if !SliceContains(&findings, f, false) {
				t.Log("missing", f)
			}
		}
		t.Log()
		//show found files that were not expected
		for _, f := range findings {
			if !SliceContains(&pathsToCheck, f, false) {
				t.Log("not expecting", f)
			}
		}
		t.Log()
		t.Error(errors.New("path counts don't match. did you add/remove something in the local templates directory?"))
	}
}

func getExpectedOutputFiles(name, templateType, provisioningMethod, tool string) []string {

	iDir := protonInfrastructureDirEnv
	if templateType == "service" {
		iDir = protonInfrastructureDirSvc
	}

	root := path.Join(name, "v1")
	schemaDir := path.Join(root, "schema")
	infraDir := path.Join(root, iDir)

	pathsToCheck := []string{
		path.Join(root, "proton.yaml"),
		path.Join(root, "README.md"),
		path.Join(schemaDir, "schema.yaml"),
		path.Join(infraDir, "manifest.yaml"),
	}

	if provisioningMethod == provisioningTypeAWSManaged {
		pathsToCheck = append(pathsToCheck, path.Join(infraDir, "cloudformation.yaml"))

	} else if provisioningMethod == provisioningTypeCodeBuild && tool == "terraform" {
		pathsToCheck = append(pathsToCheck, path.Join(infraDir, "main.tf"))
		pathsToCheck = append(pathsToCheck, path.Join(infraDir, "variables.tf"))
		pathsToCheck = append(pathsToCheck, path.Join(infraDir, "outputs.tf"))
		pathsToCheck = append(pathsToCheck, path.Join(infraDir, "output.sh"))
		pathsToCheck = append(pathsToCheck, path.Join(infraDir, "install-terraform.sh"))
	}

	return pathsToCheck
}
