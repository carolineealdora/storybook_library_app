package book

import (
	"testing"
	"time"

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
