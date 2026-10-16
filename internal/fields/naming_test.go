package fields

import "testing"

func TestPascal(t *testing.T) {
	cases := map[string]string{
		"title":       "Title",
		"author_id":   "AuthorID",
		"api_key":     "APIKey",
		"blog_post":   "BlogPost",
		"url":         "URL",
		"uuid":        "UUID",
		"http_status": "HTTPStatus",
	}
	for in, want := range cases {
		if got := Pascal(in); got != want {
			t.Errorf("Pascal(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCamel(t *testing.T) {
	cases := map[string]string{
		"title":        "title",
		"author_id":    "authorId",
		"published_at": "publishedAt",
	}
	for in, want := range cases {
		if got := Camel(in); got != want {
			t.Errorf("Camel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPlural(t *testing.T) {
	cases := map[string]string{
		"post":     "posts",
		"category": "categories",
		"box":      "boxes",
		"status":   "statuses",
		"dish":     "dishes",
		"match":    "matches",
		"day":      "days", // vowel + y
		"user":     "users",
	}
	for in, want := range cases {
		if got := Plural(in); got != want {
			t.Errorf("Plural(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHumanize(t *testing.T) {
	cases := map[string]string{
		"title":     "Title",
		"blog_post": "Blog post",
		"author_id": "Author id",
	}
	for in, want := range cases {
		if got := Humanize(in); got != want {
			t.Errorf("Humanize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPkgName(t *testing.T) {
	if got := PkgName("blog_post"); got != "blogpost" {
		t.Errorf("PkgName = %q, want blogpost", got)
	}
}
