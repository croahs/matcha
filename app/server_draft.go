package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/floatpane/matcha/config"
	"github.com/floatpane/matcha/fetcher"
	"github.com/floatpane/matcha/internal/exportcmd"
	"github.com/floatpane/matcha/tui"
	"github.com/google/uuid"
	"golang.org/x/net/html"
)

// isDraftsFolder reports whether a server folder holds drafts, such as
// "Drafts", "INBOX.Drafts" or Gmail's "[Gmail]/Drafts".
func isDraftsFolder(name string) bool {
	return strings.Contains(strings.ToLower(name), "draft")
}

// loadServerDraftCmd turns a server draft into a composer draft. Its file
// attachments are fetched and written to a temporary folder so the composer
// can attach them again.
func loadServerDraftCmd(account config.Account, email fetcher.Email, folder string) tea.Cmd {
	return func() tea.Msg {
		ref := tui.ServerDraftRef{AccountID: email.AccountID, Folder: folder, UID: email.UID}
		var paths []string
		var dir string
		for _, att := range email.Attachments {
			if att.Inline || att.IsSMIMESignature || att.IsPGPSignature || att.IsCalendarInvite {
				continue
			}
			data := att.Data
			if len(data) == 0 {
				var err error
				data, err = fetcher.FetchFolderAttachment(&account, folder, email.UID, att.PartID, att.Encoding)
				if err != nil {
					return tui.ServerDraftReadyMsg{Err: fmt.Errorf("fetch attachment %s: %w", att.Filename, err)}
				}
			}
			if dir == "" {
				var err error
				if dir, err = os.MkdirTemp("", "matcha-draft-"); err != nil {
					return tui.ServerDraftReadyMsg{Err: err}
				}
			}
			path := filepath.Join(dir, exportcmd.SanitizeFilename(att.Filename))
			if err := os.WriteFile(path, data, 0o600); err != nil {
				return tui.ServerDraftReadyMsg{Err: err}
			}
			paths = append(paths, path)
		}

		body := email.Body
		if email.BodyMIMEType == "text/html" || (email.BodyMIMEType == "" && looksLikeHTML(body)) {
			body = htmlToPlainText(body)
		}

		return tui.ServerDraftReadyMsg{
			Ref: ref,
			Draft: config.Draft{
				ID:              uuid.New().String(),
				To:              strings.Join(email.To, ", "),
				Subject:         email.Subject,
				Body:            body,
				AttachmentPaths: paths,
				AccountID:       email.AccountID,
				FromOverride:    email.From,
				InReplyTo:       email.InReplyTo,
				References:      email.References,
			},
		}
	}
}

func looksLikeHTML(s string) bool {
	t := strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(t, "<html") || strings.HasPrefix(t, "<!doctype") || strings.HasPrefix(t, "<div") || strings.HasPrefix(t, "<p")
}

// blockTags start and end on a line of their own.
var blockTags = map[string]bool{
	"div": true, "p": true, "li": true, "ul": true, "ol": true, "tr": true, "table": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true, "blockquote": true, "pre": true,
}

// htmlToPlainText keeps the text and line breaks of an HTML body, which is
// what the composer edits; formatting is dropped.
func htmlToPlainText(body string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return body
	}
	var b strings.Builder
	newline := func() {
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
			b.WriteString("\n")
		}
	}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.TextNode:
			b.WriteString(n.Data)
			return
		case html.ElementNode:
			switch n.Data {
			case "br":
				b.WriteString("\n")
				return
			case "script", "style", "head":
				return
			}
		}
		block := n.Type == html.ElementNode && blockTags[n.Data]
		if block {
			newline()
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if block {
			newline()
		}
	}
	for _, n := range doc.Nodes {
		walk(n)
	}
	return strings.TrimSpace(b.String())
}

// deleteServerDraftAfterSend removes the server copy of a draft that was just
// sent from the composer. wait runs the delete before returning, for when the
// program is about to exit.
func (m *Model) deleteServerDraftAfterSend(wait bool) {
	ref := m.pendingServerDraft
	m.pendingServerDraft = nil
	if ref == nil || m.service == nil {
		return
	}
	m.store.RemoveFolderEmail(ref.Folder, ref.UID, ref.AccountID)
	if m.folderInbox != nil && m.folderName() == ref.Folder {
		m.folderInbox.GetInbox().RemoveEmail(ref.UID, ref.AccountID)
	}
	svc := m.service
	del := func() {
		if err := svc.DeleteEmails(ref.AccountID, ref.Folder, []uint32{ref.UID}); err != nil {
			log.Printf("Error deleting server draft after send: %v", err)
		}
	}
	if wait {
		del()
		return
	}
	go del()
}
