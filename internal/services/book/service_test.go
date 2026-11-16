package book

import (
	"testing"
	"time"

	respConst "codebase/internal/constant/response"
	"codebase/models"
)

func TestAccess_PRDOrder(t *testing.T) {
	cases := []struct {
		price             int
		owned, subscribed bool
		want              string
	}{
		{0, false, false, models.AccessFree},
		{0, true, true, models.AccessFree}, // free wins: never shows a lock or "Subscribed"
		{15000, true, true, models.AccessOwned},
		{15000, false, true, models.AccessSubscribed},
		{15000, false, false, models.AccessLocked},
	}
	for _, c := range cases {
		if got := Access(c.price, c.owned, c.subscribed); got != c.want {
			t.Errorf("Access(%d, %v, %v) = %s, want %s", c.price, c.owned, c.subscribed, got, c.want)
		}
	}
}

func TestVisible_OwnersKeepDelistedBooks(t *testing.T) {
	deleted := time.Now()
	cases := []struct {
		name string
		b    *models.Book
		want bool
	}{
		{"missing", nil, false},
		{"published", &models.Book{Status: models.BookPublished}, true},
		{"draft", &models.Book{Status: models.BookDraft}, false},
		{"deleted", &models.Book{Status: models.BookPublished, DeletedAt: &deleted}, false},
		{"deleted but owned", &models.Book{Status: models.BookPublished, DeletedAt: &deleted, Owned: true}, true},
		{"draft but owned", &models.Book{Status: models.BookDraft, Owned: true}, true},
	}
	for _, c := range cases {
		if got := Visible(c.b); got != c.want {
			t.Errorf("%s: Visible = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAgeLabel(t *testing.T) {
	three := 3
	if got := AgeLabel(models.AgeRange, 1, &three); got != "1–3" {
		t.Errorf("range label = %q", got)
	}
	if got := AgeLabel(models.AgePlus, 3, nil); got != "3+" {
		t.Errorf("plus label = %q", got)
	}
}

func TestValidateInput(t *testing.T) {
	png := &Upload{FileName: "cover.PNG", Content: []byte("\x89PNG\r\n\x1a\n0000")}
	epub := &Upload{FileName: "story.epub", Content: []byte("PK\x03\x04mimetypeapplication/epub+zip")}
	valid := func() Input {
		return Input{Title: " Kancil ", Author: "A", Description: "D", AgeType: "range", AgeMin: "1", AgeMax: "3",
			Price: "15000", Status: "published", Cover: png, Epub: epub}
	}

	cases := []struct {
		name     string
		mutate   func(*Input)
		wantCode string // "" = valid
	}{
		{"valid range", func(*Input) {}, ""},
		{"valid plus, free", func(in *Input) { in.AgeType, in.AgeMax, in.Price = "plus", "", "0" }, ""},
		{"files optional (update)", func(in *Input) { in.Cover, in.Epub = nil, nil }, ""},
		{"min equals max", func(in *Input) { in.AgeMax = "1" }, respConst.INVALID_AGE_RANGE},
		{"min above max", func(in *Input) { in.AgeMin = "5" }, respConst.INVALID_AGE_RANGE},
		{"decimal age", func(in *Input) { in.AgeMin = "1.5" }, respConst.INVALID_AGE_RANGE},
		{"negative age", func(in *Input) { in.AgeMin = "-1" }, respConst.INVALID_AGE_RANGE},
		{"plus with max", func(in *Input) { in.AgeType = "plus" }, respConst.INVALID_AGE_RANGE},
		{"unknown age type", func(in *Input) { in.AgeType = "number" }, respConst.INVALID_AGE_RANGE},
		{"negative price", func(in *Input) { in.Price = "-1" }, respConst.VALIDATION_ERROR},
		{"decimal price", func(in *Input) { in.Price = "10.5" }, respConst.VALIDATION_ERROR},
		{"blank title", func(in *Input) { in.Title = "  " }, respConst.VALIDATION_ERROR},
		{"bad status", func(in *Input) { in.Status = "archived" }, respConst.VALIDATION_ERROR},
		{"gif cover", func(in *Input) { in.Cover = &Upload{FileName: "c.gif", Content: []byte("GIF89a")} }, respConst.INVALID_FILE_FORMAT},
		{"png bytes named .txt", func(in *Input) { in.Cover = &Upload{FileName: "c.txt", Content: png.Content} }, respConst.INVALID_FILE_FORMAT},
		{"pdf as book", func(in *Input) { in.Epub = &Upload{FileName: "b.pdf", Content: []byte("%PDF-1.4")} }, respConst.INVALID_FILE_FORMAT},
		{".epub that is not a zip", func(in *Input) { in.Epub = &Upload{FileName: "b.epub", Content: []byte("hello")} }, respConst.INVALID_FILE_FORMAT},
	}
	for _, c := range cases {
		in := valid()
		c.mutate(&in)
		w, ie := ValidateInput(in)
		gotCode := ""
		if ie != nil {
			gotCode = ie.code
		}
		if gotCode != c.wantCode {
			t.Errorf("%s: code = %q, want %q (%v)", c.name, gotCode, c.wantCode, ie)
			continue
		}
		if c.name == "valid range" && (w.Title != "Kancil" || *w.AgeMax != 3 || w.Cover.MimeType != "image/png") {
			t.Errorf("valid range: unexpected write %+v", w)
		}
	}
}
