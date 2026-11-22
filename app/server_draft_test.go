package app

import "testing"

func TestHTMLToPlainText(t *testing.T) {
	in := `<html><body><div dir="auto">Hei!<br/><br/>Linje to.<br/><br/>Vennlig hilsen<br/>Peder</div></body></html>`
	want := "Hei!\n\nLinje to.\n\nVennlig hilsen\nPeder"
	if got := htmlToPlainText(in); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestIsDraftsFolder(t *testing.T) {
	for name, want := range map[string]bool{"[Gmail]/Drafts": true, "INBOX.Drafts": true, "INBOX": false, "[Gmail]/Sent Mail": false} {
		if got := isDraftsFolder(name); got != want {
			t.Errorf("%s: got %v", name, got)
		}
	}
}
