package inject

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "routes.tsx")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const tsx = `<Routes>
  <Route element={<App />}>
    <Route index element={<Home />} />
    {/* gor:routes */}
  </Route>
</Routes>
`

func TestInsertBefore(t *testing.T) {
	path := write(t, tsx)
	ok, err := InsertBefore(path, "{/* gor:routes */}", "<Route path=\"posts\" element={<PostsList />} />")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected insertion")
	}
	got := read(t, path)
	want := `    <Route path="posts" element={<PostsList />} />
    {/* gor:routes */}`
	if !strings.Contains(got, want) {
		t.Errorf("injected content not indented/placed correctly:\n%s", got)
	}
}

func TestInsertBeforeIdempotent(t *testing.T) {
	path := write(t, tsx)
	block := "<Route path=\"posts\" element={<PostsList />} />"
	if _, err := InsertBefore(path, "{/* gor:routes */}", block); err != nil {
		t.Fatal(err)
	}
	after := read(t, path)
	ok, err := InsertBefore(path, "{/* gor:routes */}", block)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("second insertion should be a no-op")
	}
	if read(t, path) != after {
		t.Error("file changed on second insertion")
	}
}

func TestInsertBeforeMissingMarker(t *testing.T) {
	path := write(t, "no markers here\n")
	_, err := InsertBefore(path, "{/* gor:routes */}", "<Route />")
	if err == nil {
		t.Fatal("expected error for missing marker")
	}
	if !strings.Contains(err.Error(), "marker") {
		t.Errorf("unhelpful error: %v", err)
	}
}

func TestInsertBeforeGoFormats(t *testing.T) {
	src := `package routes

func Register() {
	// gor:routes
}
`
	path := filepath.Join(t.TempDir(), "routes.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, err := InsertBefore(path, "// gor:routes", "post.Register(api, q)")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected insertion")
	}
	got := read(t, path)
	if !strings.Contains(got, "\tpost.Register(api, q)\n\t// gor:routes") {
		t.Errorf("unexpected result:\n%s", got)
	}
}

func TestInsertBeforeMultilineBlock(t *testing.T) {
	path := write(t, tsx)
	block := "<Route path=\"posts\">\n  <Route index element={<PostsList />} />\n</Route>"
	if _, err := InsertBefore(path, "{/* gor:routes */}", block); err != nil {
		t.Fatal(err)
	}
	got := read(t, path)
	if !strings.Contains(got, "    <Route path=\"posts\">\n      <Route index") {
		t.Errorf("nested indentation wrong:\n%s", got)
	}
}
