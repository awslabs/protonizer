package cmd

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/jritsema/scaffolder"
)

func TestGenerateEnvironmentTemplate(t *testing.T) {
	internalTestGenerateTemplate(t, templateTypeEnvironment, protonInfrastructureDirEnv, tfEnvInfraSrcDir)
}

func TestGenerateServiceTemplate(t *testing.T) {
	internalTestGenerateTemplate(t, templateTypeService, protonInfrastructureDirSvc, tfSvcInfraSrcDir)
}

func internalTestGenerateTemplate(t *testing.T, templateType protonTemplateType, infraDir, infraSrcDir string) {
	verbose = true

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
	err = generateTemplate(input)
	if err != nil {
		t.Error(err)
	}

	err = scaffolder.InspectFS(destFS, t.Log, false)
	if err != nil {
		t.Error(err)
	}

	//prepend template name to output directories
	schemaDir := path.Join(name, protonSchemaDir)
	infraDir = path.Join(name, infraDir)
	infraSrcDir = path.Join(name, infraSrcDir)

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

	//add user files
	for file := range userFiles {
		pathsToCheck = append(pathsToCheck, path.Join(infraSrcDir, file))
	}

	t.Log("pathsToCheck")
	t.Log(pathsToCheck)

	findings := 0
	err = hackpadfs.WalkDir(destFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		t.Log("looking for path in destFS", path)
		if SliceContains(&pathsToCheck, path) {
			findings++
			t.Log("found", path)
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
