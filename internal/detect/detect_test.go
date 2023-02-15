package detect

import (
	"testing"

	"github.com/b4b4r07/changed-objects/internal/git"
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
			name: "terraform",
			changes: []git.Change{
				{Path: "terraform/service-a/prod/a.tf", Type: git.Addition},
				{Path: "terraform/service-a/prod/b.tf", Type: git.Addition},
				{Path: "terraform/service-a/dev/a.tf", Type: git.Addition},
				{Path: "terraform/service-a/dev/b.tf", Type: git.Addition},
				{Path: "terraform/service-b/prod/a.tf", Type: git.Addition},
				{Path: "terraform/service-b/prod/b.tf", Type: git.Addition},
				{Path: "terraform/service-b/dev/a.tf", Type: git.Addition},
				{Path: "terraform/service-b/dev/b.tf", Type: git.Addition},
			},
			patterns: []string{},
			want: map[string][]git.Change{
				"terraform/service-a/prod": {
					{Path: "terraform/service-a/prod/a.tf", Type: git.Addition},
					{Path: "terraform/service-a/prod/b.tf", Type: git.Addition},
				},
				"terraform/service-a/dev": {
					{Path: "terraform/service-a/dev/a.tf", Type: git.Addition},
					{Path: "terraform/service-a/dev/b.tf", Type: git.Addition},
				},
				"terraform/service-b/prod": {
					{Path: "terraform/service-b/prod/a.tf", Type: git.Addition},
					{Path: "terraform/service-b/prod/b.tf", Type: git.Addition},
				},
				"terraform/service-b/dev": {
					{Path: "terraform/service-b/dev/a.tf", Type: git.Addition},
					{Path: "terraform/service-b/dev/b.tf", Type: git.Addition},
				},
			},
		},
		{
			name: "terraform case 2",
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
					{Path: "terraform/service-a/prod/child/a.tf", Type: git.Addition},
				},
				"terraform/service-b/dev": {
					{Path: "terraform/service-b/dev/a.tf", Type: git.Addition},
					{Path: "terraform/service-b/dev/b.tf", Type: git.Addition},
				},
			},
		},
		{
			name: "kubernetes",
			changes: []git.Change{
				// {Path: "kubernetes/service-a/prod/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/dev/Deployment/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-a/dev/CronJob/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-b/base/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-b/overlays/prod/a.yaml", Type: git.Addition},
				{Path: "kubernetes/service-b/overlays/dev/a.yaml", Type: git.Addition},
			},
			patterns: []string{},
			want: map[string][]git.Change{
				// "kubernetes/service-a/prod": {
				// 	{Path: "kubernetes/service-a/prod/a.yaml", Type: git.Addition},
				// 	{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition},
				// 	{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition},
				// },
				"kubernetes/service-a/prod/Deployment": {{Path: "kubernetes/service-a/prod/Deployment/a.yaml", Type: git.Addition}},
				"kubernetes/service-a/prod/CronJob":    {{Path: "kubernetes/service-a/prod/CronJob/a.yaml", Type: git.Addition}},
				"kubernetes/service-a/dev/Deployment":  {{Path: "kubernetes/service-a/dev/Deployment/a.yaml", Type: git.Addition}},
				"kubernetes/service-a/dev/CronJob":     {{Path: "kubernetes/service-a/dev/CronJob/a.yaml", Type: git.Addition}},
				"kubernetes/service-b/base":            {{Path: "kubernetes/service-b/base/a.yaml", Type: git.Addition}},
				"kubernetes/service-b/overlays/dev":    {{Path: "kubernetes/service-b/overlays/dev/a.yaml", Type: git.Addition}},
				"kubernetes/service-b/overlays/prod":   {{Path: "kubernetes/service-b/overlays/prod/a.yaml", Type: git.Addition}},
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := findDirWithPatterns(tt.changes, tt.patterns)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("User value is mismatch (-got +want):\n%s", diff)
			}
		})
	}
}
