package detect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/babarot/changed-objects/internal/git"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/k0kubun/pp/v3"
)

func Test_findDirWithPatterns(t *testing.T) {
	cases := []struct {
		name     string
		changes  []git.Change
		patterns []string
		want     map[string][]git.Change
	}{
		{
			name: "terraform: no patterns case",
			changes: []git.Change{
				{Path: "terraform/service-a/prod/a.tf", Type: git.Addition},
				{Path: "terraform/service-a/prod/b.tf", Type: git.Addition},
				{Path: "terraform/service-a/dev/a.tf", Type: git.Addition},
				{Path: "terraform/service-b/prod/a.tf", Type: git.Addition},
			},
			patterns: []string{},
			want: map[string][]git.Change{
				"terraform/service-a/prod": {
					{Path: "terraform/service-a/prod/a.tf", Type: git.Addition},
					{Path: "terraform/service-a/prod/b.tf", Type: git.Addition},
				},
				"terraform/service-a/dev": {
					{Path: "terraform/service-a/dev/a.tf", Type: git.Addition},
				},
				"terraform/service-b/prod": {
					{Path: "terraform/service-b/prod/a.tf", Type: git.Addition},
				},
			},
		},
		{
			name: "terraform: including child dir with no patterns",
			changes: []git.Change{
				{Path: "terraform/service-a/prod/a.tf", Type: git.Addition},
				{Path: "terraform/service-a/prod/child/a.tf", Type: git.Addition},
				{Path: "terraform/service-b/dev/a.tf", Type: git.Addition},
				{Path: "terraform/service-b/dev/b.tf", Type: git.Addition},
			},
			patterns: []string{},
			want: map[string][]git.Change{
				"terraform/service-a/prod": {
					{Path: "terraform/service-a/prod/a.tf", Type: git.Addition},
				},
				"terraform/service-a/prod/child": {
					{Path: "terraform/service-a/prod/child/a.tf", Type: git.Addition},
				},
				"terraform/service-b/dev": {
					{Path: "terraform/service-b/dev/a.tf", Type: git.Addition},
					{Path: "terraform/service-b/dev/b.tf", Type: git.Addition},
				},
			},
		},
		{
			name: "kubernetes: no patterns case",
			changes: []git.Change{
				{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/dev/Deployment/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/dev/CronJob/a.yaml", Type: git.Addition},
			},
			patterns: []string{},
			want: map[string][]git.Change{
				"kubernetes/service-a/prod/Deployment": {{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition}},
				"kubernetes/service-a/prod/CronJob":    {{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition}},
				"kubernetes/service-a/dev/Deployment":  {{Path: "kubernetes/service-a/dev/Deployment/a.yaml", Type: git.Addition}},
				"kubernetes/service-a/dev/CronJob":     {{Path: "kubernetes/service-a/dev/CronJob/a.yaml", Type: git.Addition}},
			},
		},
		{
			name: "mixed paths: no patterns",
			changes: []git.Change{
				{Path: "kubernetes/service-a/prod/README.md", Type: git.Addition},
				{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition},
			},
			patterns: []string{},
			want: map[string][]git.Change{
				"kubernetes/service-a/prod":            {{Path: "kubernetes/service-a/prod/README.md", Type: git.Addition}},
				"kubernetes/service-a/prod/Deployment": {{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition}},
				"kubernetes/service-a/prod/CronJob":    {{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition}},
			},
		},
		{
			name: "kubernetes: pattern match",
			changes: []git.Change{
				{Path: "kubernetes/service-a/prod/README.md", Type: git.Addition},
				{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition},
			},
			// min match: if patterns are passed
			patterns: []string{"kubernetes/**/prod"},
			want: map[string][]git.Change{
				"kubernetes/service-a/prod": {
					{Path: "kubernetes/service-a/prod/README.md", Type: git.Addition},
					{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition},
					{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition},
				},
			},
		},
		{
			name: "kubernetes: different pattern match",
			changes: []git.Change{
				{Path: "kubernetes/service-a/prod/README.md", Type: git.Addition},
				{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition},
			},
			// min match: if patterns are passed
			patterns: []string{"kubernetes/*"},
			want: map[string][]git.Change{
				"kubernetes/service-a": {
					{Path: "kubernetes/service-a/prod/README.md", Type: git.Addition},
					{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition},
					{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition},
				},
			},
		},
		{
			name: "kubernetes: root pattern match",
			changes: []git.Change{
				{Path: "kubernetes/service-a/prod/README.md", Type: git.Addition},
				{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition},
			},
			// min match: if patterns are passed
			patterns: []string{"kubernetes/**"},
			want: map[string][]git.Change{
				"kubernetes": {
					{Path: "kubernetes/service-a/prod/README.md", Type: git.Addition},
					{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition},
					{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition},
				},
			},
		},
		{
			name: "kubernetes: complex pattern match",
			changes: []git.Change{
				{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/dev/Deployment/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/dev/CronJob/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-b/base/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-b/overlays/prod/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-b/overlays/dev/a.yaml", Type: git.Addition},
			},
			patterns: []string{
				"kubernetes/**/{dev,prod}",
				"kubernetes/**/overlays/{dev,prod}",
			},
			want: map[string][]git.Change{
				"kubernetes/service-a/prod": {
					{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition},
					{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition},
				},
				"kubernetes/service-a/dev": {
					{Path: "kubernetes/service-a/dev/Deployment/a.yaml", Type: git.Addition},
					{Path: "kubernetes/service-a/dev/CronJob/a.yaml", Type: git.Addition},
				},
				"kubernetes/service-b/overlays/dev":  {{Path: "kubernetes/service-b/overlays/dev/a.yaml", Type: git.Addition}},
				"kubernetes/service-b/overlays/prod": {{Path: "kubernetes/service-b/overlays/prod/a.yaml", Type: git.Addition}},
			},
		},
		{
			name: "complex org structure: no patterns",
			changes: []git.Change{
				{Path: "terraform/organizations/10x.co.jp/folders/partners/google_privileged_access_manager_entitlement.tf", Type: git.Addition},
				{Path: "terraform/organizations/10x.co.jp/google_organization_iam_custom_role.tf", Type: git.Modification},
			},
			patterns: []string{},
			want: map[string][]git.Change{
				"terraform/organizations/10x.co.jp/folders/partners": {
					{Path: "terraform/organizations/10x.co.jp/folders/partners/google_privileged_access_manager_entitlement.tf", Type: git.Addition},
				},
				"terraform/organizations/10x.co.jp": {
					{Path: "terraform/organizations/10x.co.jp/google_organization_iam_custom_role.tf", Type: git.Modification},
				},
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := findDirWithPatterns(tt.changes, tt.patterns)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Result is mismatch (-got +want):\n%s", diff)
			}
		})
	}
}

func Test_findRootByMarker(t *testing.T) {
	// Build directory structure:
	// tmpDir/
	//   terraform/services/service-a/production/
	//     main.tf          <- marker
	//     config.json
	//     scripts/
	//       deploy.sh
	//   terraform/services/service-b/
	//     main.tf          <- marker (in parent)
	//     development/
	//       config.json
	//   docs/
	//     readme.md        <- no .tf files
	//   data/
	//     schema.json      <- no .tf files
	//     nested/
	//       file.json      <- no .tf files

	tmpDir := t.TempDir()

	dirs := []string{
		"terraform/services/service-a/production/scripts",
		"terraform/services/service-b/development",
		"docs",
		"data/nested",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		"terraform/services/service-a/production/main.tf":       "",
		"terraform/services/service-a/production/config.json":   "",
		"terraform/services/service-a/production/scripts/deploy.sh": "",
		"terraform/services/service-b/main.tf":                  "",
		"terraform/services/service-b/development/config.json":  "",
		"docs/readme.md":                                         "",
		"data/schema.json":                                       "",
		"data/nested/file.json":                                  "",
	}
	for name, content := range files {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		name   string
		dir    string
		marker string
		want   string // empty string means no root found (skip)
	}{
		{
			name:   "direct parent has .tf file",
			dir:    "terraform/services/service-a/production",
			marker: "*.tf",
			want:   "terraform/services/service-a/production",
		},
		{
			name:   "child dir change, parent has .tf file",
			dir:    "terraform/services/service-a/production/scripts",
			marker: "*.tf",
			want:   "terraform/services/service-a/production",
		},
		{
			name:   "multiple levels up, grandparent has .tf file",
			dir:    "terraform/services/service-b/development",
			marker: "*.tf",
			want:   "terraform/services/service-b",
		},
		{
			name:   "no .tf files in ancestors",
			dir:    "docs",
			marker: "*.tf",
			want:   "",
		},
		{
			name:   "no .tf files, nested",
			dir:    "data/nested",
			marker: "*.tf",
			want:   "",
		},
		{
			name:   "different marker pattern: *.json",
			dir:    "terraform/services/service-a/production/scripts",
			marker: "*.json",
			want:   "terraform/services/service-a/production",
		},
		{
			name:   "different marker: *.md",
			dir:    "docs",
			marker: "*.md",
			want:   "docs",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(tmpDir, tt.dir)
			got := findRootByMarker(dir, tt.marker)
			var want string
			if tt.want != "" {
				want = filepath.Join(tmpDir, tt.want)
			}
			if got != want {
				t.Errorf("findRootByMarker(%q, %q) = %q, want %q", dir, tt.marker, got, want)
			}
		})
	}
}

func newTestClient(changes []git.Change, opt Option) client {
	printer := pp.New()
	printer.SetColoringEnabled(false)
	printer.SetExportedOnly(true)
	return client{
		opt:     opt,
		changes: changes,
		pp:      printer,
	}
}

func Test_getFile(t *testing.T) {
	t.Parallel()
	change := git.Change{Path: "terraform/service-a/main.tf", Type: git.Addition}
	got := getFile(change)

	if got.Name != "main.tf" {
		t.Errorf("Name = %q, want %q", got.Name, "main.tf")
	}
	if got.Path != "terraform/service-a/main.tf" {
		t.Errorf("Path = %q, want %q", got.Path, "terraform/service-a/main.tf")
	}
	if got.Type != git.Addition {
		t.Errorf("Type = %v, want %v", got.Type, git.Addition)
	}
	if got.ParentDir.Path != "terraform/service-a" {
		t.Errorf("ParentDir.Path = %q, want %q", got.ParentDir.Path, "terraform/service-a")
	}
}

func Test_getFiles(t *testing.T) {
	t.Parallel()
	changes := []git.Change{
		{Path: "a/b.tf", Type: git.Addition},
		{Path: "c/d.tf", Type: git.Deletion},
	}
	c := newTestClient(changes, Option{})
	files := c.getFiles(changes)

	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0].Name != "b.tf" {
		t.Errorf("files[0].Name = %q, want %q", files[0].Name, "b.tf")
	}
	if files[1].Type != git.Deletion {
		t.Errorf("files[1].Type = %v, want %v", files[1].Type, git.Deletion)
	}
}

func Test_getFiles_empty(t *testing.T) {
	t.Parallel()
	c := newTestClient(nil, Option{})
	files := c.getFiles(nil)
	if files != nil {
		t.Errorf("expected nil for empty changes, got %v", files)
	}
}

func Test_getDirs(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a directory that "exists"
	existDir := filepath.Join(tmpDir, "service-a")
	os.MkdirAll(existDir, 0755)

	changes := []git.Change{
		{Path: filepath.Join(tmpDir, "service-a", "main.tf"), Type: git.Addition},
		{Path: filepath.Join(tmpDir, "service-a", "vars.tf"), Type: git.Modification},
	}

	c := newTestClient(changes, Option{})
	dirs := c.getDirs(changes)

	if len(dirs) != 1 {
		t.Fatalf("len(dirs) = %d, want 1", len(dirs))
	}
	if dirs[0].Path != filepath.Join(tmpDir, "service-a") {
		t.Errorf("dir.Path = %q, want %q", dirs[0].Path, filepath.Join(tmpDir, "service-a"))
	}
	if len(dirs[0].Files) != 2 {
		t.Errorf("len(dir.Files) = %d, want 2", len(dirs[0].Files))
	}
	if !dirs[0].Exist {
		t.Error("dir.Exist = false, want true")
	}
}

func Test_getDirs_withRootMarker(t *testing.T) {
	tmpDir := t.TempDir()
	// Create structure: tmpDir/svc/env/scripts/
	// marker file in tmpDir/svc/env/
	svcEnv := filepath.Join(tmpDir, "svc", "env")
	scripts := filepath.Join(svcEnv, "scripts")
	os.MkdirAll(scripts, 0755)
	os.WriteFile(filepath.Join(svcEnv, "main.tf"), []byte(""), 0644)

	changes := []git.Change{
		{Path: filepath.Join(scripts, "deploy.sh"), Type: git.Addition},
	}

	c := newTestClient(changes, Option{RootMarker: "*.tf"})
	dirs := c.getDirs(changes)

	if len(dirs) != 1 {
		t.Fatalf("len(dirs) = %d, want 1", len(dirs))
	}
	if dirs[0].Path != svcEnv {
		t.Errorf("dir.Path = %q, want %q (resolved by root-marker)", dirs[0].Path, svcEnv)
	}
}

func Test_getDirs_rootMarkerNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	noMarkerDir := filepath.Join(tmpDir, "nomarker")
	os.MkdirAll(noMarkerDir, 0755)

	changes := []git.Change{
		{Path: filepath.Join(noMarkerDir, "file.txt"), Type: git.Addition},
	}

	c := newTestClient(changes, Option{RootMarker: "*.tf"})
	dirs := c.getDirs(changes)

	// Should be skipped because no marker found
	if len(dirs) != 0 {
		t.Errorf("len(dirs) = %d, want 0 (no root marker found)", len(dirs))
	}
}

func TestRun_FilterByArgs(t *testing.T) {
	changes := []git.Change{
		{Path: "terraform/svc-a/main.tf", Type: git.Addition},
		{Path: "kubernetes/svc-b/deploy.yaml", Type: git.Addition},
	}
	c := newTestClient(changes, Option{})
	c.args = []string{"terraform"}

	diff, err := c.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(diff.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(diff.Files))
	}
	if diff.Files[0].Path != "terraform/svc-a/main.tf" {
		t.Errorf("Files[0].Path = %q, want terraform/svc-a/main.tf", diff.Files[0].Path)
	}
}

func TestRun_FilterByIgnore(t *testing.T) {
	changes := []git.Change{
		{Path: "terraform/svc-a/main.tf", Type: git.Addition},
		{Path: "terraform/modules/mod-a/main.tf", Type: git.Addition},
	}
	c := newTestClient(changes, Option{Ignores: []string{"terraform/modules/**"}})

	diff, err := c.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(diff.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(diff.Files))
	}
	if diff.Files[0].Path != "terraform/svc-a/main.tf" {
		t.Errorf("Files[0].Path = %q, want terraform/svc-a/main.tf", diff.Files[0].Path)
	}
}

func TestRun_FilterByType(t *testing.T) {
	changes := []git.Change{
		{Path: "a/add.tf", Type: git.Addition},
		{Path: "a/del.tf", Type: git.Deletion},
		{Path: "a/mod.tf", Type: git.Modification},
	}
	c := newTestClient(changes, Option{Types: []string{"added", "modified"}})

	diff, err := c.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(diff.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(diff.Files))
	}
}

func TestRun_FilterByDirExist(t *testing.T) {
	tmpDir := t.TempDir()
	existDir := filepath.Join(tmpDir, "exists")
	os.MkdirAll(existDir, 0755)

	changes := []git.Change{
		{Path: filepath.Join(existDir, "a.tf"), Type: git.Addition},
		{Path: filepath.Join(tmpDir, "gone", "b.tf"), Type: git.Addition},
	}

	// dir-exist=true: only existing dirs
	c := newTestClient(changes, Option{DirExist: "true"})
	diff, err := c.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(diff.Files) != 1 {
		t.Fatalf("DirExist=true: len(Files) = %d, want 1", len(diff.Files))
	}

	// dir-exist=false: only non-existing dirs
	c2 := newTestClient(changes, Option{DirExist: "false"})
	diff2, err := c2.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(diff2.Files) != 1 {
		t.Fatalf("DirExist=false: len(Files) = %d, want 1", len(diff2.Files))
	}

	// dir-exist=all: return all
	c3 := newTestClient(changes, Option{DirExist: "all"})
	diff3, err := c3.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(diff3.Files) != 2 {
		t.Fatalf("DirExist=all: len(Files) = %d, want 2", len(diff3.Files))
	}
}

func TestRun_EmptyChanges(t *testing.T) {
	t.Parallel()
	c := newTestClient(nil, Option{})

	diff, err := c.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(diff.Files) != 0 {
		t.Errorf("len(Files) = %d, want 0", len(diff.Files))
	}
	if len(diff.Dirs) != 0 {
		t.Errorf("len(Dirs) = %d, want 0", len(diff.Dirs))
	}
}

func TestRun_GroupBy(t *testing.T) {
	changes := []git.Change{
		{Path: "k8s/svc-a/prod/Deployment/a.yaml", Type: git.Addition},
		{Path: "k8s/svc-a/prod/CronJob/b.yaml", Type: git.Addition},
	}
	c := newTestClient(changes, Option{GroupBy: []string{"k8s/**/prod"}})

	diff, err := c.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	sortDirs := cmpopts.SortSlices(func(a, b Dir) bool { return a.Path < b.Path })
	if len(diff.Dirs) != 1 {
		t.Fatalf("len(Dirs) = %d, want 1", len(diff.Dirs))
	}
	_ = sortDirs // available if needed
	if diff.Dirs[0].Path != "k8s/svc-a/prod" {
		t.Errorf("Dirs[0].Path = %q, want k8s/svc-a/prod", diff.Dirs[0].Path)
	}
	if len(diff.Dirs[0].Files) != 2 {
		t.Errorf("len(Dirs[0].Files) = %d, want 2", len(diff.Dirs[0].Files))
	}
}

func Test_getSteps(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path string
		want []string
	}{
		{"a/b/c", []string{"a/b/c", "a/b", "a"}},
		{"a", []string{"a"}},
		{"a/b", []string{"a/b", "a"}},
	}
	for _, tt := range cases {
		got := getSteps(tt.path)
		if diff := cmp.Diff(got, tt.want); diff != "" {
			t.Errorf("getSteps(%q): (-got +want):\n%s", tt.path, diff)
		}
	}
}
