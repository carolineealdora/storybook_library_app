//go:build integration

// End-to-end tests over HTTP against a real Postgres. They DROP the public schema, so they
// only run when DB_NAME ends in "_test":
//
//	DB_NAME=storybook_test go test -tags integration ./handler/
package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"codebase/internal/services/subscription"
	"codebase/models"

	"golang.org/x/crypto/bcrypt"
)

var holder models.SharedHolder

func TestMain(m *testing.M) {
	if !strings.HasSuffix(os.Getenv("DB_NAME"), "_test") {
		fmt.Println("skipping integration tests: DB_NAME must end with _test")
		os.Exit(0)
	}
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "integration-test-secret")
	}
	if err := os.Chdir(".."); err != nil { // repo root, so SCHEMA_DIR's default resolves
		panic(err)
	}
	holder.Init()
	if err := holder.Sql.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		panic(err)
	}
	if err := applySchema(); err != nil {
		panic(err)
	}
	if err := seedFixtures(); err != nil {
		panic(err)
	}
	InitRoute(&holder)
	os.Exit(m.Run())
}

// Test-only accounts; never used outside this suite.
const (
	parentEmail  = "parent@mainstory.test"
	parent2Email = "parent2@mainstory.test"
	adminEmail   = "admin@mainstory.test"
	parentPass   = "it-parent-pass"
	adminPass    = "it-admin-pass"
)

// applySchema runs the shared schema owned by storybook_management_app (its ADR-010).
// Filenames start with a timestamp, so sorted order is apply order.
func applySchema() error {
	dir := os.Getenv("SCHEMA_DIR")
	if dir == "" {
		dir = "../storybook_management_app/migrations"
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.sql")) // Glob returns names sorted
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no .sql files in SCHEMA_DIR %q", dir)
	}
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if err := holder.Sql.Exec(string(content)); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

// seedFixtures creates two parents, one admin and the monthly plan, hashing passwords at run time.
func seedFixtures() error {
	parentHash, err := bcrypt.GenerateFromPassword([]byte(parentPass), bcrypt.MinCost)
	if err != nil {
		return err
	}
	adminHash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.MinCost)
	if err != nil {
		return err
	}
	for _, p := range []struct{ name, email string }{{"Demo Parent", parentEmail}, {"Second Parent", parent2Email}} {
		if err := holder.Sql.Exec(`WITH u AS (INSERT INTO users (name) VALUES (?) RETURNING id)
			INSERT INTO credentials (email, password_hash, account_type, user_id) SELECT ?, ?, 'parent', id FROM u`,
			p.name, p.email, string(parentHash)); err != nil {
			return err
		}
	}
	if err := holder.Sql.Exec(`WITH a AS (INSERT INTO admins (name) VALUES ('Admin') RETURNING id)
		INSERT INTO credentials (email, password_hash, account_type, admin_id) SELECT ?, ?, 'admin', id FROM a`,
		adminEmail, string(adminHash)); err != nil {
		return err
	}
	return holder.Sql.Exec(`INSERT INTO subscription_plans (code, name, price, benefits)
		VALUES ('monthly', 'Monthly Subscription', 49000, 'Read all published books. New books every month. Cancel anytime.')`)
}

// ---- tiny HTTP client ----

type result struct {
	Status int
	Data   json.RawMessage
	Meta   struct{ Total, Page, PageSize int }
	Error  struct{ Code, Message string }
	Body   []byte
	Header http.Header
}

func send(t *testing.T, req *http.Request, token string) result {
	t.Helper()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	holder.Echo.ServeHTTP(rec, req)
	r := result{Status: rec.Code, Body: rec.Body.Bytes(), Header: rec.Header()}
	if strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		var env struct {
			Data json.RawMessage
			Meta struct {
				Total, Page int
				PageSize    int `json:"page_size"`
			}
			Error struct{ Code, Message string }
		}
		if err := json.Unmarshal(r.Body, &env); err != nil {
			t.Fatalf("bad json %s: %v", r.Body, err)
		}
		r.Data, r.Error = env.Data, env.Error
		r.Meta.Total, r.Meta.Page, r.Meta.PageSize = env.Meta.Total, env.Meta.Page, env.Meta.PageSize
	}
	return r
}

func call(t *testing.T, method, path, token string, body any) result {
	t.Helper()
	return parentCall(t, method, path, token, "", body)
}

// parentCall sends X-Parent-Token too: the proof of Parent view that payment and settings routes need (TD-002).
func parentCall(t *testing.T, method, path, token, parentToken string, body any) result {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, "/api/v1"+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if parentToken != "" {
		req.Header.Set("X-Parent-Token", parentToken)
	}
	return send(t, req, token)
}

func form(t *testing.T, method, path, token string, fields map[string]string, files map[string][2]string) result {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		w.WriteField(k, v)
	}
	for field, f := range files { // f = {filename, content}
		fw, _ := w.CreateFormFile(field, f[0])
		fw.Write([]byte(f[1]))
	}
	w.Close()
	req := httptest.NewRequest(method, "/api/v1"+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return send(t, req, token)
}

func decode[T any](t *testing.T, r result) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(r.Data, &v); err != nil {
		t.Fatalf("decode %s: %v", r.Data, err)
	}
	return v
}

func expect(t *testing.T, r result, status int, code string) {
	t.Helper()
	if r.Status != status || r.Error.Code != code {
		t.Fatalf("got %d %q (%s), want %d %q", r.Status, r.Error.Code, r.Body, status, code)
	}
}

func login(t *testing.T, path, email, password string) (access, refresh, parentToken string) {
	t.Helper()
	r := call(t, "POST", path, "", map[string]string{"email": email, "password": password})
	expect(t, r, 200, "")
	tok := decode[tokens](t, r)
	return tok.AccessToken, tok.RefreshToken, tok.ParentToken
}

func epubBytes(t *testing.T) string {
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	f, _ := z.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	f.Write([]byte("application/epub+zip"))
	z.Close()
	return buf.String()
}

type bookDTO struct {
	ID, Access string
	AgeLabel   string `json:"age_label"`
	Price      int
}
type orderDTO struct {
	ID, Status string
	Price      int
}

// meDTO is GET /me as far as CR-001 cares: the badge and the pending-price notice.
type meDTO struct {
	PlanStatus   string `json:"plan_status"`
	IsSubscriber bool   `json:"is_subscriber"`
	Subscription *struct {
		ID                 string
		PricePaid          int  `json:"price_paid"`
		NextRenewalPrice   int  `json:"next_renewal_price"`
		PriceChangePending bool `json:"price_change_pending"`
	}
}

type sub struct {
	PricePaid  int `json:"price_paid"`
	ID, Status string
	OrderType  *string   `json:"order_type"`
	StartedAt  time.Time `json:"started_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ParentToken  string `json:"parent_token"`
}

func findBook(books []bookDTO, id string) *bookDTO {
	for i := range books {
		if books[i].ID == id {
			return &books[i]
		}
	}
	return nil
}

// ---- the story ----

func TestLibraryEndToEnd(t *testing.T) {
	cover := [2]string{"cover.png", "\x89PNG\r\n\x1a\n-fake-png-body"}
	epub := [2]string{"story.epub", epubBytes(t)}

	// US-01 / US-15 / E17 / E19
	expect(t, call(t, "POST", "/auth/login", "", map[string]string{"email": parentEmail, "password": "nope"}), 401, "INVALID_CREDENTIALS")
	expect(t, call(t, "POST", "/auth/login", "", map[string]string{"email": "ghost@x.test", "password": "nope"}), 401, "INVALID_CREDENTIALS")
	expect(t, call(t, "POST", "/admin/auth/login", "", map[string]string{"email": parentEmail, "password": parentPass}), 403, "FORBIDDEN")
	parent, parentRefresh, pt := login(t, "/auth/login", "PARENT@mainstory.test", parentPass) // email is case-insensitive
	parent2, _, pt2 := login(t, "/auth/login", parent2Email, parentPass)
	admin, adminRefresh, adminPT := login(t, "/admin/auth/login", adminEmail, adminPass)
	if pt == "" || adminPT != "" {
		t.Fatalf("parent login must return a parent_token, admin login must not (got %q / %q)", pt, adminPT)
	}
	expect(t, call(t, "GET", "/books", "", nil), 401, "UNAUTHORIZED")
	expect(t, call(t, "GET", "/books", "garbage", nil), 401, "UNAUTHORIZED")
	expect(t, call(t, "GET", "/admin/books", parent, nil), 403, "FORBIDDEN")
	expect(t, call(t, "GET", "/books", admin, nil), 403, "FORBIDDEN")
	expect(t, call(t, "GET", "/nope", parent, nil), 404, "NOT_FOUND")

	// US-16: admin creates books; E15 / E16 validation
	newBook := func(title, price, status string) string {
		r := form(t, "POST", "/admin/books", admin, map[string]string{"title": title, "author": "Kak Seto", "description": "A story",
			"age_type": "range", "age_min": "1", "age_max": "3", "price": price, "status": status},
			map[string][2]string{"cover": cover, "epub": epub})
		expect(t, r, 200, "")
		return decode[bookDTO](t, r).ID
	}
	free := newBook("Free Book", "0", "published")
	bookA := newBook("Book A", "15000", "published")
	bookB := newBook("Book B", "20000", "published")
	draft := newBook("Draft Book", "10000", "draft")
	bad := func(fields map[string]string, files map[string][2]string, code string) {
		t.Helper()
		base := map[string]string{"title": "X", "author": "Y", "description": "Z", "age_type": "range", "age_min": "1", "age_max": "3", "price": "1", "status": "draft"}
		for k, v := range fields {
			base[k] = v
		}
		if files == nil {
			files = map[string][2]string{"cover": cover, "epub": epub}
		}
		expect(t, form(t, "POST", "/admin/books", admin, base, files), 400, code)
	}
	bad(nil, map[string][2]string{"cover": {"c.gif", "GIF89a..."}, "epub": epub}, "INVALID_FILE_FORMAT")
	bad(nil, map[string][2]string{"cover": cover, "epub": {"b.pdf", "%PDF-1.4"}}, "INVALID_FILE_FORMAT")
	bad(map[string]string{"age_min": "5", "age_max": "3"}, nil, "INVALID_AGE_RANGE")
	bad(map[string]string{"age_min": "1.5"}, nil, "INVALID_AGE_RANGE")
	bad(map[string]string{"price": "-1"}, nil, "VALIDATION_ERROR")
	bad(nil, map[string][2]string{"cover": cover}, "VALIDATION_ERROR") // epub missing
	plus := form(t, "PUT", "/admin/books/"+draft, admin, map[string]string{"title": "Draft Book", "author": "A", "description": "D",
		"age_type": "plus", "age_min": "3", "price": "10000", "status": "draft"}, nil)
	expect(t, plus, 200, "")
	if got := decode[bookDTO](t, plus).AgeLabel; got != "3+" {
		t.Fatalf("age label = %q, want 3+", got)
	}
	adminList := decode[[]bookDTO](t, call(t, "GET", "/admin/books?status=draft&search=draft", admin, nil))
	if len(adminList) != 1 || adminList[0].ID != draft {
		t.Fatalf("admin search/filter = %+v", adminList)
	}
	if r := call(t, "GET", "/admin/books/"+draft+"/cover", admin, nil); r.Status != 200 || r.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("admin cover: %d %s", r.Status, r.Header.Get("Content-Type"))
	}

	// US-04: showcase. Drafts hidden; access per book.
	books := decode[[]bookDTO](t, call(t, "GET", "/books", parent, nil))
	if len(books) != 3 || findBook(books, draft) != nil {
		t.Fatalf("showcase should hold 3 published books, got %+v", books)
	}
	if findBook(books, free).Access != "free" || findBook(books, bookA).Access != "locked" || findBook(books, bookA).AgeLabel != "1–3" {
		t.Fatalf("showcase access wrong: %+v", books)
	}
	// TD-003: pagination
	page1 := call(t, "GET", "/books?page_size=2", parent, nil)
	if n := len(decode[[]bookDTO](t, page1)); n != 2 || page1.Meta.Total != 3 || page1.Meta.Page != 1 || page1.Meta.PageSize != 2 {
		t.Fatalf("page 1: %d books, meta %+v", n, page1.Meta)
	}
	page2 := call(t, "GET", "/books?page=2&page_size=2", parent, nil)
	if p2 := decode[[]bookDTO](t, page2); len(p2) != 1 || findBook(decode[[]bookDTO](t, page1), p2[0].ID) != nil {
		t.Fatalf("page 2 = %+v", p2)
	}
	if r := call(t, "GET", "/books", parent, nil); r.Meta.PageSize != 20 || r.Meta.Total != 3 {
		t.Fatalf("default meta = %+v", r.Meta)
	}
	expect(t, call(t, "GET", "/books?page_size=101", parent, nil), 400, "VALIDATION_ERROR")
	expect(t, call(t, "GET", "/books?page=-1", parent, nil), 400, "VALIDATION_ERROR")
	if r := call(t, "GET", "/admin/books?page_size=1", admin, nil); len(decode[[]bookDTO](t, r)) != 1 || r.Meta.Total != 4 {
		t.Fatalf("admin page = %s meta %+v", r.Data, r.Meta)
	}
	expect(t, call(t, "GET", "/books/"+draft, parent, nil), 404, "BOOK_NOT_FOUND") // E22
	expect(t, call(t, "GET", "/books/not-a-uuid", parent, nil), 400, "VALIDATION_ERROR")

	// US-07 / E18: content gate
	expect(t, call(t, "GET", "/books/"+bookA+"/content", parent, nil), 403, "BOOK_LOCKED")
	if r := call(t, "GET", "/books/"+free+"/content", parent, nil); r.Status != 200 || string(r.Body) != epub[1] || r.Header.Get("Content-Type") != "application/epub+zip" {
		t.Fatalf("free content: %d %s", r.Status, r.Header.Get("Content-Type"))
	}
	if r := call(t, "GET", "/books/"+bookA+"/cover", parent, nil); r.Status != 200 || string(r.Body) != cover[1] {
		t.Fatalf("cover of locked book should be visible: %d", r.Status)
	}

	// TD-002: payment and settings need proof of Parent view (X-Parent-Token), not just the session
	buyA := map[string]string{"book_id": bookA, "idempotency_key": "k-gate"}
	expect(t, call(t, "POST", "/book-orders", parent, buyA), 403, "PARENT_MODE_REQUIRED")               // Child view: no parent token
	expect(t, parentCall(t, "POST", "/book-orders", parent, pt2, buyA), 403, "PARENT_MODE_REQUIRED")    // someone else's
	expect(t, parentCall(t, "POST", "/book-orders", parent, parent, buyA), 403, "PARENT_MODE_REQUIRED") // access token is not one
	expect(t, call(t, "POST", "/subscription-orders", parent, map[string]string{"idempotency_key": "s-gate"}), 403, "PARENT_MODE_REQUIRED")
	expect(t, call(t, "POST", "/subscriptions/current/cancel", parent, nil), 403, "PARENT_MODE_REQUIRED")
	expect(t, call(t, "PATCH", "/me", parent, map[string]string{"name": "Kid"}), 403, "PARENT_MODE_REQUIRED")
	expect(t, call(t, "PUT", "/me/pin", parent, map[string]string{"pin": "1111", "pin_confirmation": "1111"}), 403, "PARENT_MODE_REQUIRED")
	expect(t, call(t, "GET", "/me", pt, nil), 401, "UNAUTHORIZED") // parent token is not an access token

	// US-09: buy a single book
	expect(t, parentCall(t, "POST", "/book-orders", parent, pt, map[string]string{"book_id": free, "idempotency_key": "k-free"}), 409, "BOOK_NOT_PURCHASABLE") // E10
	expect(t, parentCall(t, "POST", "/book-orders", parent, pt, map[string]string{"book_id": draft, "idempotency_key": "k-draft"}), 404, "BOOK_NOT_FOUND")
	r := parentCall(t, "POST", "/book-orders", parent, pt, map[string]string{"book_id": bookA, "idempotency_key": "k-a1"})
	expect(t, r, 200, "")
	orderA := decode[orderDTO](t, r)
	if orderA.Status != "pending" || orderA.Price != 15000 {
		t.Fatalf("order A = %+v", orderA)
	}
	expect(t, parentCall(t, "POST", "/book-orders", parent, pt, map[string]string{"book_id": bookA, "idempotency_key": "k-a1"}), 409, "DUPLICATE_REQUEST") // E8
	// E7: admin raises the price while the parent is on checkout
	expect(t, form(t, "PUT", "/admin/books/"+bookA, admin, map[string]string{"title": "Book A", "author": "Kak Seto", "description": "A story",
		"age_type": "range", "age_min": "1", "age_max": "3", "price": "99000", "status": "published"}, nil), 200, "")
	expect(t, parentCall(t, "POST", "/book-orders/"+orderA.ID+"/pay", parent2, pt2, map[string]string{"result": "success"}), 404, "ORDER_NOT_FOUND") // not yours
	expect(t, parentCall(t, "POST", "/book-orders/"+orderA.ID+"/pay", parent, pt, map[string]string{"result": "maybe"}), 400, "VALIDATION_ERROR")
	r = parentCall(t, "POST", "/book-orders/"+orderA.ID+"/pay", parent, pt, map[string]string{"result": "success"})
	expect(t, r, 200, "")
	if paid := decode[orderDTO](t, r); paid.Status != "paid" || paid.Price != 15000 {
		t.Fatalf("paid order = %+v, want price 15000 snapshot", paid)
	}
	expect(t, parentCall(t, "POST", "/book-orders/"+orderA.ID+"/pay", parent, pt, map[string]string{"result": "success"}), 409, "DUPLICATE_REQUEST")        // E8 double tap
	expect(t, parentCall(t, "POST", "/book-orders", parent, pt, map[string]string{"book_id": bookA, "idempotency_key": "k-a2"}), 409, "BOOK_ALREADY_OWNED") // E1
	expect(t, call(t, "GET", "/books/"+bookA+"/content", parent, nil), 200, "")
	expect(t, call(t, "GET", "/books/"+bookA+"/content", parent2, nil), 403, "BOOK_LOCKED")

	// E9: failed payment keeps the book locked; retry with a new order
	r = parentCall(t, "POST", "/book-orders", parent, pt, map[string]string{"book_id": bookB, "idempotency_key": "k-b1"})
	expect(t, r, 200, "")
	orderB := decode[orderDTO](t, r)
	expect(t, parentCall(t, "POST", "/book-orders/"+orderB.ID+"/pay", parent, pt, map[string]string{"result": "failed"}), 402, "PAYMENT_FAILED")
	expect(t, call(t, "GET", "/books/"+bookB+"/content", parent, nil), 403, "BOOK_LOCKED")
	// E6: book unpublished while on checkout
	r = parentCall(t, "POST", "/book-orders", parent, pt, map[string]string{"book_id": bookB, "idempotency_key": "k-b2"})
	expect(t, r, 200, "")
	orderB2 := decode[orderDTO](t, r)
	unpublishB := func(status string) {
		expect(t, form(t, "PUT", "/admin/books/"+bookB, admin, map[string]string{"title": "Book B", "author": "Kak Seto", "description": "A story",
			"age_type": "range", "age_min": "1", "age_max": "3", "price": "20000", "status": status}, nil), 200, "")
	}
	unpublishB("draft")
	expect(t, parentCall(t, "POST", "/book-orders/"+orderB2.ID+"/pay", parent, pt, map[string]string{"result": "success"}), 409, "BOOK_UNAVAILABLE")
	unpublishB("published")

	// US-12: order history = finished book orders, newest first, price paid
	historyResp := call(t, "GET", "/book-orders?page_size=1", parent, nil)
	if h := decode[[]orderDTO](t, historyResp); len(h) != 1 || h[0].Status != "failed" || historyResp.Meta.Total != 2 {
		t.Fatalf("order history page = %+v meta %+v", h, historyResp.Meta)
	}
	history := decode[[]orderDTO](t, call(t, "GET", "/book-orders", parent, nil))
	if len(history) != 2 || history[0].Status != "failed" || history[1].Status != "paid" || history[1].Price != 15000 {
		t.Fatalf("order history = %+v", history)
	}

	// US-10: subscribe. Failed payment creates nothing; success unlocks everything.
	plan := decode[struct {
		Price      int
		Code, Name string
	}](t, call(t, "GET", "/subscription/plan", parent, nil))
	if plan.Code != "monthly" || plan.Name == "" {
		t.Fatalf("plan = %+v", plan)
	}
	// CR-001 T2 / B4: owning a purchased book does not make the account "subscribed"
	if me := decode[meDTO](t, call(t, "GET", "/me", parent, nil)); me.PlanStatus != "free" || me.IsSubscriber || me.Subscription != nil {
		t.Fatalf("buyer without subscription: %+v", me)
	}
	expect(t, parentCall(t, "POST", "/subscription-orders", parent, pt, map[string]any{"idempotency_key": "s-0"}), 400, "VALIDATION_ERROR") // expected_price required
	r = parentCall(t, "POST", "/subscription-orders", parent, pt, map[string]any{"idempotency_key": "s-1", "expected_price": plan.Price})
	expect(t, r, 200, "")
	subOrder1 := decode[orderDTO](t, r)
	if subOrder1.Price != plan.Price {
		t.Fatalf("subscription order price %d, plan %d", subOrder1.Price, plan.Price)
	}
	expect(t, parentCall(t, "POST", "/subscription-orders/"+subOrder1.ID+"/pay", parent, pt, map[string]string{"result": "failed"}), 402, "PAYMENT_FAILED")
	if me := decode[meDTO](t, call(t, "GET", "/me", parent, nil)); me.IsSubscriber || me.PlanStatus != "free" {
		t.Fatal("failed payment must not subscribe")
	}
	r = parentCall(t, "POST", "/subscription-orders", parent, pt, map[string]any{"idempotency_key": "s-2", "expected_price": plan.Price})
	expect(t, r, 200, "")
	subOrder2 := decode[orderDTO](t, r)
	r = parentCall(t, "POST", "/subscription-orders/"+subOrder2.ID+"/pay", parent, pt, map[string]string{"result": "success"})
	expect(t, r, 200, "")
	first := decode[sub](t, r)
	if first.Status != "active" || first.ExpiresAt.Sub(first.StartedAt) < 28*24*time.Hour || first.PricePaid != plan.Price {
		t.Fatalf("subscription = %+v", first)
	}
	// CR-001 T1: badge flips to subscribed right after payment; no price change pending yet
	if me := decode[meDTO](t, call(t, "GET", "/me", parent, nil)); me.PlanStatus != "subscribed" || !me.IsSubscriber ||
		me.Subscription == nil || me.Subscription.PriceChangePending || me.Subscription.NextRenewalPrice != plan.Price {
		t.Fatalf("subscriber me = %+v", me)
	}
	books = decode[[]bookDTO](t, call(t, "GET", "/books", parent, nil))
	if findBook(books, bookB).Access != "subscribed" || findBook(books, bookA).Access != "owned" || findBook(books, free).Access != "free" {
		t.Fatalf("subscriber showcase = %+v", books)
	}
	expect(t, call(t, "GET", "/books/"+bookB+"/content", parent, nil), 200, "")
	expect(t, parentCall(t, "POST", "/book-orders", parent, pt, map[string]string{"book_id": bookB, "idempotency_key": "k-b3"}), 409, "SUBSCRIPTION_ACTIVE")                 // E2
	expect(t, parentCall(t, "POST", "/book-orders/"+orderB2.ID+"/pay", parent, pt, map[string]string{"result": "success"}), 409, "SUBSCRIPTION_ACTIVE")                      // pending order, now subscribed
	expect(t, parentCall(t, "POST", "/subscription-orders", parent, pt, map[string]any{"idempotency_key": "s-3", "expected_price": plan.Price}), 409, "SUBSCRIPTION_ACTIVE") // E3

	// US-17 / CR-001-2: a plan price change applies to new subscriptions and the next renewal only
	setPlan := func(token string, price any) result {
		return call(t, "PUT", "/admin/subscription-plan", token, map[string]any{"price": price, "benefits": "All books"})
	}
	expect(t, setPlan(admin, 0), 400, "INVALID_PRICE")                                                                     // T8 / E-CR2
	expect(t, setPlan(admin, -5), 400, "INVALID_PRICE")                                                                    // T8
	expect(t, call(t, "PUT", "/admin/subscription-plan", admin, map[string]any{"benefits": "x"}), 400, "VALIDATION_ERROR") // price missing
	expect(t, setPlan(parent, 59000), 403, "FORBIDDEN")                                                                    // T9 / E-CR3
	expect(t, setPlan(admin, 59000), 200, "")                                                                              // P1
	if r := setPlan(admin, 59000); r.Status != 200 || decode[struct{ Price int }](t, r).Price != 59000 {                   // E-CR6: same price is a no-op
		t.Fatalf("same-price update: %d %s", r.Status, r.Body)
	}
	expect(t, setPlan(admin, 69000), 200, "") // second change before the renewal (P5 / E-CR4)
	type change struct {
		OldPrice  int    `json:"old_price"`
		NewPrice  int    `json:"new_price"`
		ChangedBy string `json:"changed_by_name"`
	}
	hist := call(t, "GET", "/admin/subscription-plan/history", admin, nil)
	if h := decode[[]change](t, hist); hist.Meta.Total != 2 || h[0].OldPrice != 59000 || h[0].NewPrice != 69000 ||
		h[1].OldPrice != plan.Price || h[1].NewPrice != 59000 || h[0].ChangedBy == "" {
		t.Fatalf("price history = %+v meta %+v", h, hist.Meta)
	}
	expect(t, call(t, "GET", "/admin/subscription-plan/history", parent, nil), 403, "FORBIDDEN")
	// T4 / P2: the running period keeps its price; /me announces the next renewal price
	if me := decode[meDTO](t, call(t, "GET", "/me", parent, nil)); me.Subscription == nil || me.Subscription.ID != first.ID ||
		me.Subscription.PricePaid != plan.Price || me.Subscription.NextRenewalPrice != 69000 || !me.Subscription.PriceChangePending {
		t.Fatalf("subscriber after price change: %+v", me)
	}

	// US-13: auto-renew creates a new active period and expires the old one
	if err := holder.Sql.Exec(`UPDATE subscriptions SET started_at = now() - interval '1 month 1 minute', expires_at = now() - interval '1 minute' WHERE id = ?`, first.ID); err != nil {
		t.Fatal(err)
	}
	renewer := subscription.NewService(&holder)
	if n, err := renewer.RenewDue(); err != nil || n != 1 {
		t.Fatalf("RenewDue = %d, %v", n, err)
	}
	if n, err := renewer.RenewDue(); err != nil || n != 0 {
		t.Fatalf("second RenewDue = %d, %v (must be idempotent)", n, err)
	}
	periods := decode[[]sub](t, call(t, "GET", "/subscriptions", parent, nil))
	if len(periods) != 2 || periods[0].Status != "active" || *periods[0].OrderType != "renewal" ||
		periods[1].Status != "expired" || *periods[1].OrderType != "new" || !periods[0].StartedAt.Equal(periods[1].ExpiresAt) ||
		periods[0].PricePaid != 69000 || periods[1].PricePaid != plan.Price { // T5 / E-CR4: renewal charges the latest price
		t.Fatalf("subscription history = %+v", periods)
	}
	if me := decode[meDTO](t, call(t, "GET", "/me", parent, nil)); me.Subscription == nil || me.Subscription.PriceChangePending {
		t.Fatalf("pending flag must clear after renewal: %+v", me)
	}

	// US-11 / E4 / E14: cancel locks non-owned books immediately, purchases stay
	r = parentCall(t, "POST", "/subscriptions/current/cancel", parent, pt, nil)
	expect(t, r, 200, "")
	if c := decode[sub](t, r); c.Status != "cancelled" {
		t.Fatalf("cancelled = %+v", c)
	}
	expect(t, parentCall(t, "POST", "/subscriptions/current/cancel", parent, pt, nil), 409, "SUBSCRIPTION_NOT_ACTIVE")
	expect(t, call(t, "GET", "/books/"+bookB+"/content", parent, nil), 403, "BOOK_LOCKED")
	if n, _ := renewer.RenewDue(); n != 0 {
		t.Fatal("cancelled subscription must not renew")
	}
	if me := decode[meDTO](t, call(t, "GET", "/me", parent, nil)); me.PlanStatus != "free" || me.Subscription != nil { // B3
		t.Fatalf("cancelled me = %+v", me)
	}

	// CR-001 T7 / E-CR1: the paywall showed an old price -> 409, the app re-reads the plan
	expect(t, parentCall(t, "POST", "/subscription-orders", parent2, pt2, map[string]any{"idempotency_key": "p2-s1", "expected_price": plan.Price}), 409, "PRICE_CHANGED")
	// T6 / P3: a new subscriber pays the new price
	r = parentCall(t, "POST", "/subscription-orders", parent2, pt2, map[string]any{"idempotency_key": "p2-s2", "expected_price": 69000})
	expect(t, r, 200, "")
	p2order := decode[orderDTO](t, r)
	if p2order.Price != 69000 {
		t.Fatalf("new subscriber order = %+v", p2order)
	}
	r = parentCall(t, "POST", "/subscription-orders/"+p2order.ID+"/pay", parent2, pt2, map[string]string{"result": "success"})
	expect(t, r, 200, "")
	if s := decode[sub](t, r); s.PricePaid != 69000 {
		t.Fatalf("new subscriber period = %+v", s)
	}
	// T3: an expired subscription shows "free"
	holder.Sql.Exec(`UPDATE subscriptions SET status = 'expired' WHERE id = ?`, decode[sub](t, r).ID)
	if me := decode[meDTO](t, call(t, "GET", "/me", parent2, nil)); me.PlanStatus != "free" || me.IsSubscriber {
		t.Fatalf("expired me = %+v", me)
	}

	// US-08 / E5: soft delete hides from the shelf, owners keep it
	expect(t, call(t, "DELETE", "/admin/books/"+bookA, admin, nil), 200, "")
	expect(t, call(t, "DELETE", "/admin/books/"+bookA, admin, nil), 404, "BOOK_NOT_FOUND")
	if findBook(decode[[]bookDTO](t, call(t, "GET", "/books", parent, nil)), bookA) != nil {
		t.Fatal("deleted book still on the shelf")
	}
	mine := decode[[]bookDTO](t, call(t, "GET", "/me/books", parent, nil))
	if len(mine) != 1 || mine[0].ID != bookA || mine[0].Access != "owned" {
		t.Fatalf("my books = %+v", mine)
	}
	expect(t, call(t, "GET", "/books/"+bookA+"/content", parent, nil), 200, "")
	expect(t, call(t, "GET", "/books/"+bookA, parent2, nil), 404, "BOOK_NOT_FOUND")
	if mine2 := decode[[]bookDTO](t, call(t, "GET", "/me/books", parent2, nil)); len(mine2) != 0 {
		t.Fatalf("parent2 my books = %+v", mine2)
	}

	// US-14: settings
	expect(t, parentCall(t, "PATCH", "/me", parent, pt, map[string]string{"name": "   "}), 400, "VALIDATION_ERROR")
	expect(t, parentCall(t, "PATCH", "/me", parent, pt, map[string]string{"name": "Bunda"}), 200, "")

	// US-03 / E11 / E12 / E13: PIN
	expect(t, call(t, "POST", "/me/pin/verify", parent, map[string]string{"pin": "1234"}), 409, "PIN_NOT_SET")
	expect(t, parentCall(t, "PUT", "/me/pin", parent, pt, map[string]string{"pin": "1234", "pin_confirmation": "4321"}), 400, "VALIDATION_ERROR")
	expect(t, parentCall(t, "PUT", "/me/pin", parent, pt, map[string]string{"pin": "12a4", "pin_confirmation": "12a4"}), 400, "VALIDATION_ERROR")
	expect(t, parentCall(t, "PUT", "/me/pin", parent, pt, map[string]string{"pin": "1234", "pin_confirmation": "1234"}), 200, "")
	expect(t, parentCall(t, "PUT", "/me/pin", parent, pt, map[string]string{"pin": "5678", "pin_confirmation": "5678"}), 401, "INVALID_PASSWORD")
	expect(t, parentCall(t, "PUT", "/me/pin", parent, pt, map[string]string{"pin": "5678", "pin_confirmation": "5678", "password": parentPass}), 200, "")
	me := decode[struct {
		Name   string
		PinSet bool `json:"pin_set"`
	}](t, call(t, "GET", "/me", parent, nil))
	if me.Name != "Bunda" || !me.PinSet {
		t.Fatalf("me = %+v", me)
	}
	// A correct PIN or password hands back a fresh parent token (Child -> Parent view)
	r = call(t, "POST", "/me/pin/verify", parent, map[string]string{"pin": "5678"})
	expect(t, r, 200, "")
	pinPT := decode[tokens](t, r).ParentToken
	expect(t, parentCall(t, "PATCH", "/me", parent, pinPT, map[string]string{"name": "Bunda"}), 200, "")

	// TD-006: Forgot PIN? allows 4 wrong passwords, then locks for 1 minute. Success resets the count.
	forgot := func(password string) result {
		return call(t, "POST", "/me/pin/forgot", parent, map[string]string{"password": password})
	}
	for i := 1; i <= 3; i++ {
		expect(t, forgot("wrong"), 401, "INVALID_PASSWORD")
	}
	r = forgot(parentPass)
	expect(t, r, 200, "")
	expect(t, parentCall(t, "PATCH", "/me", parent, decode[tokens](t, r).ParentToken, map[string]string{"name": "Bunda"}), 200, "")
	for i := 1; i <= 3; i++ {
		expect(t, forgot("wrong"), 401, "INVALID_PASSWORD")
	}
	expect(t, forgot("wrong"), 429, "TOO_MANY_ATTEMPTS")
	expect(t, forgot(parentPass), 429, "TOO_MANY_ATTEMPTS") // locked even if right
	holder.Sql.Exec(`UPDATE users SET password_locked_until = now() - interval '1 second'`)
	expect(t, forgot(parentPass), 200, "")

	// E12: 5 wrong PINs lock the PIN for 1 minute
	for i := 1; i <= 4; i++ {
		expect(t, call(t, "POST", "/me/pin/verify", parent, map[string]string{"pin": "0000"}), 401, "INVALID_PIN")
	}
	expect(t, call(t, "POST", "/me/pin/verify", parent, map[string]string{"pin": "0000"}), 429, "TOO_MANY_ATTEMPTS")
	expect(t, call(t, "POST", "/me/pin/verify", parent, map[string]string{"pin": "5678"}), 429, "TOO_MANY_ATTEMPTS") // locked even if right
	expect(t, forgot(parentPass), 200, "")                                                                           // E13: the way out of a locked PIN
	holder.Sql.Exec(`UPDATE users SET pin_locked_until = now() - interval '1 second'`)                               // a minute passes
	expect(t, call(t, "POST", "/me/pin/verify", parent, map[string]string{"pin": "5678"}), 200, "")

	// US-01 / US-02 / E21: refresh, logout, expiry
	r = call(t, "POST", "/auth/refresh", "", map[string]string{"refresh_token": parentRefresh})
	expect(t, r, 200, "")
	if refreshed := decode[tokens](t, r); refreshed.ParentToken != "" {
		t.Fatal("refresh must not grant Parent view: the refresh token is still held in Child view")
	} else {
		expect(t, call(t, "GET", "/me", refreshed.AccessToken, nil), 200, "")
	}
	expect(t, call(t, "POST", "/auth/logout", parent, map[string]string{"refresh_token": parentRefresh}), 200, "")
	expect(t, call(t, "POST", "/auth/refresh", "", map[string]string{"refresh_token": parentRefresh}), 401, "UNAUTHORIZED")
	expect(t, call(t, "POST", "/admin/auth/logout", admin, map[string]string{"refresh_token": adminRefresh}), 200, "")
	_, oldRefresh, _ := login(t, "/auth/login", parentEmail, parentPass)
	holder.Sql.Exec(`UPDATE refresh_tokens SET expires_at = now() - interval '1 second' WHERE revoked_at IS NULL`)
	expect(t, call(t, "POST", "/auth/refresh", "", map[string]string{"refresh_token": oldRefresh}), 401, "REFRESH_TOKEN_EXPIRED")
}
