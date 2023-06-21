package cmd

import (
	"errors"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/jritsema/scaffolder"
	"gopkg.in/yaml.v3"
)

func TestGenerateEnvironmentTemplate(t *testing.T) {
	internalTestGenerateTemplate(t, "environment", protonInfrastructureDirEnv, tfEnvInfraSrcDir)
}

// tests that reserved variables are mapped properly
func TestGenerateEnvironmentTemplate_ReservedVar(t *testing.T) {

	result := internalTestGenerateTemplate(t, "environment", protonInfrastructureDirEnv, tfEnvInfraSrcDir)

	f, err := result.Open("my_template/v1/infrastructure/main.tf")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	contents := string(b)
	t.Log(contents)
	if strings.Contains(contents, "name = var.environment.inputs.name") {
		t.Error("name variable should not be mapped to environment.inputs")
	}

	f, err = result.Open("my_template/v1/schema/schema.yaml")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
	b, err = ioutil.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	contents = string(b)
	t.Log(contents)
	lines := strings.Split(contents, "\n")
	if SliceContains(&lines, "title: name", true) {
		t.Error("schema should not contain variable `name`")
	}
}

// tests that reserved variables are mapped properly
func TestGenerateServiceTemplate_ReservedVar(t *testing.T) {

	result := internalTestGenerateTemplate(t, "service", protonInfrastructureDirSvc, tfSvcInfraSrcDir)

	f, err := result.Open("my_template/v1/instance_infrastructure/main.tf")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	contents := string(b)
	t.Log(contents)
	if strings.Contains(contents, "name = var.service_instance.inputs.name") {
		t.Error("name variable should not be mapped to service_instance.inputs")
	}
	if strings.Contains(contents, "environment = var.service_instance.inputs.environment") {
		t.Error("environment variable should not be mapped to service_instance.inputs")
	}

	f, err = result.Open("my_template/v1/schema/schema.yaml")
	if err != nil {
		t.Error(err)
	}
	defer f.Close()
	b, err = ioutil.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	contents = string(b)
	t.Log(contents)
	lines := strings.Split(contents, "\n")
	if SliceContains(&lines, "title: name", true) {
		t.Error("schema should not contain variable `name`")
	}
	if SliceContains(&lines, "title: environment", true) {
		t.Error("schema should not contain variable `environment`")
	}
}

func TestGenerateServiceTemplate(t *testing.T) {
	internalTestGenerateTemplate(t, "service", protonInfrastructureDirSvc, tfSvcInfraSrcDir)
}

func internalTestGenerateTemplate(t *testing.T, templateType string, infraDir, infraSrcDir string) hackpadfs.FS {

	//create in memory file system for testing
	srcFS, err := mem.NewFS()
	if err != nil {
		t.Error(err)
	}

	//populate source fs with user content
	userFiles := scaffolder.FSContents{
		"source1.tf":      []byte(""),
		"dir1/source1.tf": []byte(""),
		"dir2/source1.tf": []byte(""),
		"dir2/source2.tf": []byte(""),
	}
	err = scaffolder.PopulateFS(srcFS, userFiles)
	if err != nil {
		t.Error(err)
	}

	//create destination file system (in-memory)
	destFS, err := mem.NewFS()
	if err != nil {
		t.Error(err)
	}
	workDir, _ := os.Getwd()

	//test generateTemplate (in-memory)
	name := "my_template"
	input := generateInput{
		name:         name,
		templateType: templateType,
		srcDir:       path.Join(workDir, "test"),
		srcFS:        srcFS,
		destFS:       destFS,
	}
	err = generateCodeBuildTerraformTemplate(input)
	if err != nil {
		t.Error(err)
	}

	err = scaffolder.InspectFS(destFS, t.Log, false)
	if err != nil {
		t.Error(err)
	}

	pathsToCheck := getExpectedOutputFiles(name, templateType, "codebuild", "terraform")

	//add user files
	for file := range userFiles {
		pathsToCheck = append(pathsToCheck, path.Join(name, "v1", infraSrcDir, file))
	}

	t.Log("pathsToCheck")
	t.Log(pathsToCheck)

	found := 0
	findings := []string{}
	err = hackpadfs.WalkDir(destFS, ".", func(path string, d fs.DirEntry, err error) error {
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
		if strings.HasSuffix(path, "yaml") || strings.HasSuffix(path, "yml") {
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

	return destFS
}
