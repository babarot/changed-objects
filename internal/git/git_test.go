package git

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestType_String(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ty   Type
		want string
	}{
		{Addition, "added"},
		{Deletion, "deleted"},
		{Modification, "modified"},
		{Unknown, "unknown"},
		{Type(99), "unknown"},
	}
	for _, tt := range cases {
		if got := tt.ty.String(); got != tt.want {
			t.Errorf("Type(%d).String() = %q, want %q", tt.ty, got, tt.want)
		}
	}
}

func TestType_MarshalJSON(t *testing.T) {
	t.Parallel()
	cases := []struct {
		ty   Type
		want string
	}{
		{Addition, `"added"`},
		{Deletion, `"deleted"`},
		{Modification, `"modified"`},
		{Unknown, `"unknown"`},
	}
	for _, tt := range cases {
		b, err := json.Marshal(tt.ty)
		if err != nil {
			t.Fatalf("Marshal Type(%d): %v", tt.ty, err)
		}
		if string(b) != tt.want {
			t.Errorf("Marshal Type(%d) = %s, want %s", tt.ty, b, tt.want)
		}
	}
}

// initTestRepo creates a temporary git repo with an initial commit and returns
// the repo path. It also creates a second commit so that previousCommit() works.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}

	// first commit
	writeFile(t, dir, "a.txt", "hello")
	gitAdd(t, dir, ".")
	gitCommit(t, dir, "first commit")

	// second commit with an added file
	writeFile(t, dir, "b.txt", "world")
	gitAdd(t, dir, ".")
	gitCommit(t, dir, "second commit")

	return dir
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func gitAdd(t *testing.T, dir, target string) {
	t.Helper()
	cmd := exec.Command("git", "add", target)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %s: %v", out, err)
	}
}

func gitCommit(t *testing.T, dir, msg string) {
	t.Helper()
	cmd := exec.Command("git", "commit", "-m", msg)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %s: %v", out, err)
	}
}

func TestOpen_DefaultBranch(t *testing.T) {
	dir := initTestRepo(t)

	changes, err := Open(Config{
		Path:          dir,
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// On the default branch, it compares HEAD vs HEAD^.
	// Second commit added b.txt.
	if len(changes) == 0 {
		t.Fatal("expected at least one change, got 0")
	}

	found := false
	for _, c := range changes {
		if c.Path == "b.txt" && c.Type == Addition {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Addition of b.txt, got %+v", changes)
	}
}

func TestOpen_InvalidPath(t *testing.T) {
	_, err := Open(Config{
		Path:          "/nonexistent/path",
		DefaultBranch: "main",
	})
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

func TestOpen_WithModification(t *testing.T) {
	dir := initTestRepo(t)

	// Create a third commit that modifies a.txt
	writeFile(t, dir, "a.txt", "hello modified")
	gitAdd(t, dir, ".")
	gitCommit(t, dir, "third commit")

	changes, err := Open(Config{
		Path:          dir,
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	found := false
	for _, c := range changes {
		if c.Path == "a.txt" && c.Type == Modification {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Modification of a.txt, got %+v", changes)
	}
}

func TestOpen_WithDeletion(t *testing.T) {
	dir := initTestRepo(t)

	// Remove b.txt and commit
	os.Remove(filepath.Join(dir, "b.txt"))
	gitAdd(t, dir, ".")
	gitCommit(t, dir, "delete b.txt")

	changes, err := Open(Config{
		Path:          dir,
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	found := false
	for _, c := range changes {
		if c.Path == "b.txt" && c.Type == Deletion {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Deletion of b.txt, got %+v", changes)
	}
}

func TestGetDefaultBranch_NoRemote(t *testing.T) {
	dir := initTestRepo(t)

	cfg := Config{
		Path:          dir,
		DefaultBranch: "main",
	}

	repo, err := openPlain(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.repo = repo

	_, err = cfg.getDefaultBranch()
	if err == nil {
		t.Fatal("expected error when no remote HEAD exists")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	dir := initTestRepo(t)

	cfg := Config{
		Path:          dir,
		DefaultBranch: "main",
	}

	repo, err := openPlain(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg.repo = repo

	branch, err := cfg.getCurrentBranch()
	if err != nil {
		t.Fatalf("getCurrentBranch: %v", err)
	}
	if branch != "main" {
		t.Errorf("getCurrentBranch() = %q, want %q", branch, "main")
	}
}

func TestCurrentCommit(t *testing.T) {
	dir := initTestRepo(t)

	repo, err := openPlain(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{repo: repo}

	commit, err := cfg.currentCommit()
	if err != nil {
		t.Fatalf("currentCommit: %v", err)
	}
	if commit == nil {
		t.Fatal("expected non-nil commit")
	}
}

func TestPreviousCommit(t *testing.T) {
	dir := initTestRepo(t)

	repo, err := openPlain(dir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{repo: repo}

	commit, err := cfg.previousCommit()
	if err != nil {
		t.Fatalf("previousCommit: %v", err)
	}
	if commit == nil {
		t.Fatal("expected non-nil commit")
	}
}

func openPlain(dir string) (*gogit.Repository, error) {
	return gogit.PlainOpen(dir)
}

// --- mock repoOperator for resolveChanges tests ---

type mockRepo struct {
	branch       string
	branchErr    error
	current      *object.Commit
	currentErr   error
	previous     *object.Commit
	previousErr  error
	remotes      map[string]*object.Commit
	remoteErr    error
	mergeBaseCmt *object.Commit
	mergeBaseErr error
	defaultBr    string
	defaultBrErr error
	headBranch   string
	headBrErr    error
	changes      []Change
	changesErr   error
}

func (m *mockRepo) getCurrentBranch() (string, error)  { return m.branch, m.branchErr }
func (m *mockRepo) currentCommit() (*object.Commit, error) {
	return m.current, m.currentErr
}
func (m *mockRepo) previousCommit() (*object.Commit, error) {
	return m.previous, m.previousErr
}
func (m *mockRepo) remoteCommit(name string) (*object.Commit, error) {
	if m.remotes != nil {
		if c, ok := m.remotes[name]; ok {
			return c, m.remoteErr
		}
	}
	return nil, m.remoteErr
}
func (m *mockRepo) mergeBaseCommit(_, _ string) (*object.Commit, error) {
	return m.mergeBaseCmt, m.mergeBaseErr
}
func (m *mockRepo) getDefaultBranch() (string, error) { return m.defaultBr, m.defaultBrErr }
func (m *mockRepo) headBranchName() (string, error)   { return m.headBranch, m.headBrErr }
func (m *mockRepo) getChanges(_, _ *object.Commit) ([]Change, error) {
	return m.changes, m.changesErr
}

// sentinel commits (content doesn't matter; only used as non-nil pointers by mock)
var (
	dummyCommitA = &object.Commit{Message: "a"}
	dummyCommitB = &object.Commit{Message: "b"}
	dummyCommitC = &object.Commit{Message: "c"}
)

func TestResolveChanges_DefaultBranch(t *testing.T) {
	t.Parallel()
	want := []Change{{Path: "x.tf", Type: Addition}}
	m := &mockRepo{
		branch:   "main",
		previous: dummyCommitA,
		current:  dummyCommitB,
		changes:  want,
	}

	got, err := resolveChanges("main", "", m)
	if err != nil {
		t.Fatalf("resolveChanges: %v", err)
	}
	if len(got) != 1 || got[0].Path != "x.tf" {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveChanges_FeatureBranch(t *testing.T) {
	t.Parallel()
	want := []Change{{Path: "feat.tf", Type: Modification}}
	m := &mockRepo{
		branch:  "feature/foo",
		remotes: map[string]*object.Commit{"origin/main": dummyCommitA},
		current: dummyCommitB,
		changes: want,
	}

	got, err := resolveChanges("main", "", m)
	if err != nil {
		t.Fatalf("resolveChanges: %v", err)
	}
	if len(got) != 1 || got[0].Path != "feat.tf" {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveChanges_BaseNilFallback(t *testing.T) {
	t.Parallel()
	// Feature branch but remote returns nil → falls back to getDefaultBranch
	want := []Change{{Path: "fallback.tf", Type: Addition}}
	m := &mockRepo{
		branch:    "feature/bar",
		remotes:   map[string]*object.Commit{"origin/HEAD": dummyCommitA},
		defaultBr: "origin/HEAD",
		current:   dummyCommitB,
		changes:   want,
	}

	got, err := resolveChanges("main", "", m)
	if err != nil {
		t.Fatalf("resolveChanges: %v", err)
	}
	if len(got) != 1 || got[0].Path != "fallback.tf" {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveChanges_BaseNilFallbackError(t *testing.T) {
	t.Parallel()
	m := &mockRepo{
		branch:       "feature/baz",
		defaultBrErr: errors.New("no remote HEAD"),
		current:      dummyCommitB,
	}

	_, err := resolveChanges("main", "", m)
	if err == nil {
		t.Fatal("expected error when getDefaultBranch fails and base is nil")
	}
}

func TestResolveChanges_WithMergeBase(t *testing.T) {
	t.Parallel()
	want := []Change{{Path: "mb.tf", Type: Deletion}}
	m := &mockRepo{
		branch:       "main",
		previous:     dummyCommitA,
		current:      dummyCommitB,
		headBranch:   "main",
		mergeBaseCmt: dummyCommitC,
		changes:      want,
	}

	got, err := resolveChanges("main", "origin/main", m)
	if err != nil {
		t.Fatalf("resolveChanges: %v", err)
	}
	if len(got) != 1 || got[0].Path != "mb.tf" {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveChanges_MergeBaseNilKeepsOriginal(t *testing.T) {
	t.Parallel()
	want := []Change{{Path: "orig.tf", Type: Modification}}
	m := &mockRepo{
		branch:       "main",
		previous:     dummyCommitA,
		current:      dummyCommitB,
		headBranch:   "main",
		mergeBaseCmt: nil, // merge-base returns nil → keeps previous base
		changes:      want,
	}

	got, err := resolveChanges("main", "some-ref", m)
	if err != nil {
		t.Fatalf("resolveChanges: %v", err)
	}
	if len(got) != 1 || got[0].Path != "orig.tf" {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestResolveChanges_MergeBaseError(t *testing.T) {
	t.Parallel()
	m := &mockRepo{
		branch:       "main",
		previous:     dummyCommitA,
		current:      dummyCommitB,
		headBranch:   "main",
		mergeBaseErr: errors.New("merge-base failed"),
	}

	_, err := resolveChanges("main", "origin/main", m)
	if err == nil {
		t.Fatal("expected error when mergeBaseCommit fails")
	}
}

func TestResolveChanges_HeadBranchNameError(t *testing.T) {
	t.Parallel()
	m := &mockRepo{
		branch:   "main",
		previous: dummyCommitA,
		current:  dummyCommitB,
		headBrErr: errors.New("detached HEAD"),
	}

	_, err := resolveChanges("main", "origin/main", m)
	if err == nil {
		t.Fatal("expected error when headBranchName fails")
	}
}

func TestResolveChanges_GetCurrentBranchError(t *testing.T) {
	t.Parallel()
	m := &mockRepo{
		branchErr: errors.New("branch error"),
	}

	_, err := resolveChanges("main", "", m)
	if err == nil {
		t.Fatal("expected error when getCurrentBranch fails")
	}
}

func TestResolveChanges_PreviousCommitError(t *testing.T) {
	t.Parallel()
	m := &mockRepo{
		branch:      "main",
		previousErr: errors.New("no previous"),
	}

	_, err := resolveChanges("main", "", m)
	if err == nil {
		t.Fatal("expected error when previousCommit fails on default branch")
	}
}

func TestResolveChanges_RemoteCommitError(t *testing.T) {
	t.Parallel()
	m := &mockRepo{
		branch:    "feature/x",
		remoteErr: errors.New("remote error"),
		remotes:   map[string]*object.Commit{"origin/main": dummyCommitA},
	}

	_, err := resolveChanges("main", "", m)
	if err == nil {
		t.Fatal("expected error when remoteCommit fails on feature branch")
	}
}

func TestResolveChanges_CurrentCommitError(t *testing.T) {
	t.Parallel()
	m := &mockRepo{
		branch:     "main",
		previous:   dummyCommitA,
		currentErr: errors.New("current error"),
	}

	_, err := resolveChanges("main", "", m)
	if err == nil {
		t.Fatal("expected error when currentCommit fails")
	}
}
