package services

import "testing"

func TestLinkService_ValidateURL(t *testing.T) {
	s := &LinkServiceImpl{}
	ok := []string{"https://x.com", "http://x.com/p?q=1", "ftp://files", "mailto:a@x.com", "example.com"}
	for _, u := range ok {
		if err := s.ValidateURL(u); err != nil {
			t.Errorf("ValidateURL(%q) should be valid, got %v", u, err)
		}
	}
	bad := []string{"", "   ", "javascript:alert(1)", "noscheme"}
	for _, u := range bad {
		if err := s.ValidateURL(u); err == nil {
			t.Errorf("ValidateURL(%q) should be invalid", u)
		}
	}
}

func TestLinkService_ExtractLinksFromHTML(t *testing.T) {
	s := &LinkServiceImpl{}
	links := s.extractLinksFromHTML(`<p>see <a href="https://x.com/a">click here</a> ok</p>`)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d: %+v", len(links), links)
	}
	if links[0].URL != "https://x.com/a" || links[0].Text != "click here" {
		t.Errorf("html link wrong: %+v", links[0])
	}
}

func TestLinkService_ExtractLinksFromPlainText(t *testing.T) {
	s := &LinkServiceImpl{}
	links := s.extractLinksFromPlainText("visit https://x.com and ftp://y.org then https://x.com again")
	if len(links) != 2 { // duplicate https://x.com is de-duplicated
		t.Fatalf("expected 2 unique links, got %d: %+v", len(links), links)
	}
	if links[0].URL != "https://x.com" || links[1].URL != "ftp://y.org" {
		t.Errorf("plain-text links wrong: %+v", links)
	}
}

func TestLinkService_CategorizeLink(t *testing.T) {
	s := &LinkServiceImpl{}
	cases := map[string]string{
		"":                       "unknown",
		"mailto:a@x.com":         "email",
		"file:///tmp/x":          "file",
		"ftp://files/x":          "file",
		"https://github.com/o/r": "external",
		"https://docs.foo.com":   "external",
		"https://example.com":    "html",
	}
	for in, want := range cases {
		if got := s.categorizeLink(in); got != want {
			t.Errorf("categorizeLink(%q) = %q, want %q", in, got, want)
		}
	}
}
