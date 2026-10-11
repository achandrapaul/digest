package tui

import (
	"testing"

	"github.com/achandrapaul/digest/pkg/model"
)

func TestEveryNavKindHasAHandler(t *testing.T) {
	for kind := KindGitRepo; kind <= KindMyPR; kind++ {
		if _, known := kindHandlers[kind]; !known {
			t.Errorf("kind %d has no handler", kind)
		}
	}
}

func TestUnknownKindsFallBackToTheNoteHandler(t *testing.T) {
	unknown := NavItemKind(99)
	item := NavItem{Kind: unknown, Note: &model.Note{ID: "n1", Summary: "Fallback"}}
	if key := navItemKey(item); key != "note:99:n1" {
		t.Fatalf("key = %q", key)
	}
	if key := navItemKey(NavItem{Kind: unknown}); key != "" {
		t.Fatalf("an empty unknown item has no key, got %q", key)
	}
	header := previewHeading{title: " PREVIEW "}
	kindHandlerFor(unknown).header(Model{}, item, &header)
	if header.title != " PREVIEW NOTE " {
		t.Fatalf("header = %+v", header)
	}
	if stamp := kindHandlerFor(unknown).stamp; stamp != nil {
		t.Fatal("an unknown kind has no preview stamp")
	}
}
