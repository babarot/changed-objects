package detect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/babarot/changed-objects/internal/git"
	"github.com/google/go-cmp/cmp"
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
