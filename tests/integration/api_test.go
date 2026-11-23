package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

var baseURL = strings.TrimRight(envOr("BASE_URL", "http://localhost:8080"), "/")

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type result struct {
	status int
	body   map[string]any
}

func (r result) str(key string) string { s, _ := r.body[key].(string); return s }

func (r result) id() int64 { f, _ := r.body["id"].(float64); return int64(f) }

func call(t *testing.T, method, path, token string, payload any) result {
	t.Helper()

	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatal(err)
		}
	}

	req, err := http.NewRequest(method, baseURL+path, &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v (server jalan?)", method, path, err)
	}
	defer resp.Body.Close()

	res := result{status: resp.StatusCode, body: map[string]any{}}
	_ = json.NewDecoder(resp.Body).Decode(&res.body)
	return res
}

func expect(t *testing.T, got result, status int, code string) {
	t.Helper()
	if got.status != status {
		t.Fatalf("status = %d, want %d (body: %v)", got.status, status, got.body)
	}
	if code != "" && got.str("error_code") != code {
		t.Fatalf("error_code = %q, want %q", got.str("error_code"), code)
	}
}

func login(t *testing.T, email string) string {
	t.Helper()
	res := call(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": email, "password": "password123"})
	expect(t, res, http.StatusOK, "")
	return res.str("token")
}

func newRoom(t *testing.T, adminToken string) int64 {
	t.Helper()
	name := fmt.Sprintf("test-room-%d", time.Now().UnixNano())
	res := call(t, http.MethodPost, "/api/v1/rooms", adminToken, map[string]any{"name": name, "capacity": 6})
	expect(t, res, http.StatusCreated, "")
	return res.id()
}

func slot(from, to int) (string, string) {
	base := time.Now().UTC().Add(72 * time.Hour).Truncate(time.Hour)
	return base.Add(time.Duration(from) * time.Hour).Format(time.RFC3339),
		base.Add(time.Duration(to) * time.Hour).Format(time.RFC3339)
}

func bookingBody(roomID int64, title, start, end string) map[string]any {
	return map[string]any{"room_id": roomID, "title": title, "start_time": start, "end_time": end}
}

func TestHealthAndUnknownRoute(t *testing.T) {
	expect(t, call(t, http.MethodGet, "/healthz", "", nil), http.StatusOK, "")
	expect(t, call(t, http.MethodGet, "/api/v1/nope", "", nil), http.StatusNotFound, "NOT_FOUND")
}

func TestLogin(t *testing.T) {
	if tok := login(t, "admin@test.com"); tok == "" {
		t.Fatal("token kosong")
	}

	wrongPass := call(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "admin@test.com", "password": "salah"})
	expect(t, wrongPass, http.StatusUnauthorized, "INVALID_CREDENTIALS")

	unknown := call(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "ghost@test.com", "password": "password123"})
	expect(t, unknown, http.StatusUnauthorized, "INVALID_CREDENTIALS")

	empty := call(t, http.MethodPost, "/api/v1/auth/login", "", map[string]string{})
	expect(t, empty, http.StatusBadRequest, "VALIDATION_ERROR")
}

func TestCreateRoomPermissions(t *testing.T) {
	admin, user := login(t, "admin@test.com"), login(t, "user1@test.com")
	body := map[string]any{"name": "Ruang Rapat", "capacity": 5}

	expect(t, call(t, http.MethodPost, "/api/v1/rooms", "", body), http.StatusUnauthorized, "UNAUTHORIZED")
	expect(t, call(t, http.MethodPost, "/api/v1/rooms", "bukan.token.valid", body), http.StatusUnauthorized, "UNAUTHORIZED")
	expect(t, call(t, http.MethodPost, "/api/v1/rooms", user, body), http.StatusForbidden, "FORBIDDEN")
	expect(t, call(t, http.MethodPost, "/api/v1/rooms", admin, map[string]any{"name": "x", "capacity": 0}), http.StatusBadRequest, "VALIDATION_ERROR")

	res := call(t, http.MethodPost, "/api/v1/rooms", admin, body)
	expect(t, res, http.StatusCreated, "")
	if res.body["is_active"] != true || res.str("name") != "Ruang Rapat" {
		t.Fatalf("response tidak sesuai: %v", res.body)
	}
}

func TestBookingLifecycle(t *testing.T) {
	admin := login(t, "admin@test.com")
	budi, sari := login(t, "user1@test.com"), login(t, "user2@test.com")
	roomID := newRoom(t, admin)

	start, end := slot(1, 3)
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)

	t.Run("invalid time", func(t *testing.T) {
		expect(t, call(t, http.MethodPost, "/api/v1/bookings", budi, bookingBody(roomID, "x", end, start)), http.StatusBadRequest, "INVALID_TIME")
		expect(t, call(t, http.MethodPost, "/api/v1/bookings", budi, bookingBody(roomID, "x", past, end)), http.StatusBadRequest, "INVALID_TIME")
		expect(t, call(t, http.MethodPost, "/api/v1/bookings", budi, bookingBody(roomID, "x", "besok pagi", end)), http.StatusBadRequest, "INVALID_TIME")
	})

	t.Run("room not found", func(t *testing.T) {
		expect(t, call(t, http.MethodPost, "/api/v1/bookings", budi, bookingBody(999999999, "x", start, end)), http.StatusNotFound, "ROOM_NOT_FOUND")
	})

	var bookingID int64
	t.Run("create", func(t *testing.T) {
		res := call(t, http.MethodPost, "/api/v1/bookings", budi, bookingBody(roomID, "Daily sync", start, end))
		expect(t, res, http.StatusCreated, "")
		if res.str("status") != "CONFIRMED" || res.str("start_time") != start {
			t.Fatalf("response tidak sesuai: %v", res.body)
		}
		bookingID = res.id()
	})

	t.Run("partial overlap conflicts", func(t *testing.T) {
		s, e := slot(2, 4)
		expect(t, call(t, http.MethodPost, "/api/v1/bookings", sari, bookingBody(roomID, "bentrok", s, e)), http.StatusConflict, "ROOM_CONFLICT")
	})

	t.Run("adjacent slot is fine", func(t *testing.T) {
		s, e := slot(3, 4)
		expect(t, call(t, http.MethodPost, "/api/v1/bookings", sari, bookingBody(roomID, "lanjutan", s, e)), http.StatusCreated, "")
	})

	t.Run("room hidden from available", func(t *testing.T) {
		res := call(t, http.MethodGet, "/api/v1/rooms/available?start="+start+"&end="+end, budi, nil)
		expect(t, res, http.StatusOK, "")
		if containsRoom(res, roomID) {
			t.Fatal("room yang sudah dibooking masih muncul di /rooms/available")
		}
	})

	t.Run("cancel", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/bookings/%d/cancel", bookingID)
		expect(t, call(t, http.MethodPatch, path, sari, nil), http.StatusForbidden, "FORBIDDEN")

		res := call(t, http.MethodPatch, path, budi, nil)
		expect(t, res, http.StatusOK, "")
		if res.str("status") != "CANCELLED" {
			t.Fatalf("status = %q", res.str("status"))
		}

		expect(t, call(t, http.MethodPatch, path, budi, nil), http.StatusConflict, "BOOKING_ALREADY_CANCELLED")
		expect(t, call(t, http.MethodPatch, "/api/v1/bookings/999999999/cancel", budi, nil), http.StatusNotFound, "BOOKING_NOT_FOUND")
	})

	t.Run("slot free again after cancel", func(t *testing.T) {
		expect(t, call(t, http.MethodPost, "/api/v1/bookings", sari, bookingBody(roomID, "ambil alih", start, end)), http.StatusCreated, "")
	})
}

func TestAdminCanCancelAnyBooking(t *testing.T) {
	admin, budi := login(t, "admin@test.com"), login(t, "user1@test.com")
	roomID := newRoom(t, admin)
	s, e := slot(5, 6)

	res := call(t, http.MethodPost, "/api/v1/bookings", budi, bookingBody(roomID, "punya budi", s, e))
	expect(t, res, http.StatusCreated, "")

	expect(t, call(t, http.MethodPatch, fmt.Sprintf("/api/v1/bookings/%d/cancel", res.id()), admin, nil), http.StatusOK, "")
}

func TestConcurrentBookingSameSlot(t *testing.T) {
	admin, budi := login(t, "admin@test.com"), login(t, "user1@test.com")
	roomID := newRoom(t, admin)
	s, e := slot(8, 10)

	const n = 10
	var (
		wg      sync.WaitGroup
		ready   = make(chan struct{})
		results = make([]result, n)
	)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ready
			results[i] = call(t, http.MethodPost, "/api/v1/bookings", budi, bookingBody(roomID, fmt.Sprintf("rebutan-%d", i), s, e))
		}()
	}
	close(ready)
	wg.Wait()

	created, conflicts := 0, 0
	for _, r := range results {
		switch {
		case r.status == http.StatusCreated:
			created++
		case r.status == http.StatusConflict && r.str("error_code") == "ROOM_CONFLICT":
			conflicts++
		default:
			t.Errorf("response tak terduga: %d %v", r.status, r.body)
		}
	}
	if created != 1 || conflicts != n-1 {
		t.Fatalf("created=%d conflicts=%d, want 1 dan %d", created, conflicts, n-1)
	}
}

func containsRoom(res result, id int64) bool {
	data, _ := res.body["data"].([]any)
	for _, item := range data {
		m, _ := item.(map[string]any)
		if f, _ := m["id"].(float64); int64(f) == id {
			return true
		}
	}
	return false
}
