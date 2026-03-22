package repogov_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/nicholashoule/repogov"
)

func TestDetectPlatforms_GitHub(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := repogov.DetectPlatforms(root)
	want := []string{"github"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DetectPlatforms = %v, want %v", got, want)
	}
}

func TestDetectPlatforms_GitLab_Dir(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".gitlab"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := repogov.DetectPlatforms(root)
	want := []string{"gitlab"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DetectPlatforms = %v, want %v", got, want)
	}
}

func TestDetectPlatforms_GitLab_CIFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitlab-ci.yml"), []byte("stages:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := repogov.DetectPlatforms(root)
	want := []string{"gitlab"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DetectPlatforms = %v, want %v", got, want)
	}
}

func TestDetectPlatforms_Bitbucket(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "bitbucket-pipelines.yml"), []byte("pipelines:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := repogov.DetectPlatforms(root)
	want := []string{"bitbucket"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DetectPlatforms = %v, want %v", got, want)
	}
}

func TestDetectPlatforms_Multiple(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".gitlab"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bitbucket-pipelines.yml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	got := repogov.DetectPlatforms(root)
	want := []string{"bitbucket", "github", "gitlab"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DetectPlatforms = %v, want %v", got, want)
	}
}

func TestDetectPlatforms_None(t *testing.T) {
	root := t.TempDir()
	got := repogov.DetectPlatforms(root)
	if len(got) != 0 {
		t.Errorf("DetectPlatforms = %v, want empty", got)
	}
}

func TestDefaultGitHubLayout(t *testing.T) {
	schema := repogov.DefaultGitHubLayout()
	if schema.Root != ".github" {
		t.Errorf("Root = %q, want %q", schema.Root, ".github")
	}
	// GitHub platform layout has no required files.
	if len(schema.Required) != 0 {
		t.Errorf("Required = %v, want empty", schema.Required)
	}
	// Should have well-known optional files.
	found := false
	for _, opt := range schema.Optional {
		if opt == "CODEOWNERS" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Optional should contain CODEOWNERS")
	}
	// Should have workflows dir.
	if _, ok := schema.Dirs["workflows"]; !ok {
		t.Error("Dirs should contain workflows")
	}
}

func TestDefaultBitbucketLayout(t *testing.T) {
	schema := repogov.DefaultBitbucketLayout()
	if schema.Root != "." {
		t.Errorf("Root = %q, want %q", schema.Root, ".")
	}
	found := false
	for _, opt := range schema.Optional {
		if opt == "bitbucket-pipelines.yml" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Optional should contain bitbucket-pipelines.yml")
	}
}
