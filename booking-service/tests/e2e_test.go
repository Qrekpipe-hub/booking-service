package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/example/booking-service/internal/handler"
	"github.com/example/booking-service/internal/model"
	"github.com/example/booking-service/internal/service"
	"github.com/example/booking-service/internal/service/mocks"
)

// ── newTestServer builds a complete in-memory server ──────────────

func newTestServer(t *testing.T) (*httptest.Server, func()) {
	t.Helper()

	roomRepo     := newFakeRoomRepo()
	scheduleRepo := newFakeScheduleRepo()
	slotRepo     := newFakeSlotRepo(scheduleRepo)
	bookingRepo  := mocks.NewMockBookingRepo()

	slotGen     := service.NewSlotGenerator(slotRepo, scheduleRepo)
	authSvc     := service.NewAuthService(newFakeUserRepo(), "e2e-test-secret")
	roomSvc     := service.NewRoomService(roomRepo)
	scheduleSvc := service.NewScheduleService(scheduleRepo, roomRepo, slotGen)
	bookingSvc  := service.NewBookingService(bookingRepo, slotRepo, &mocks.MockConferenceService{})

	router := handler.NewRouter(handler.Services{
		Auth:     authSvc,
		Room:     roomSvc,
		Schedule: scheduleSvc,
		Booking:  bookingSvc,
	})

	srv := httptest.NewServer(router)
	return srv, srv.Close
}

// ── Scenario 1: create room → create schedule → create booking ────

func TestE2E_CreateRoomScheduleBooking(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	client := srv.Client()
	base := srv.URL

	// 1. Get admin token
	adminToken := dummyLogin(t, client, base, "admin")
	userToken  := dummyLogin(t, client, base, "user")

	// 2. Create a room
	roomBody := map[string]interface{}{"name": "E2E Room", "capacity": 6}
	roomResp := doRequest(t, client, "POST", base+"/rooms/create", adminToken, roomBody)
	require.Equal(t, http.StatusCreated, roomResp.StatusCode)
	var roomOut struct {
		Room struct{ ID string `json:"id"` } `json:"room"`
	}
	decodeJSON(t, roomResp, &roomOut)
	roomID := roomOut.Room.ID
	require.NotEmpty(t, roomID)

	// 3. Create a schedule (all days, so we always get slots for "today")
	tomorrow := time.Now().UTC().AddDate(0, 0, 1)
	schedBody := map[string]interface{}{
		"daysOfWeek": []int{1, 2, 3, 4, 5, 6, 7},
		"startTime":  "09:00",
		"endTime":    "18:00",
	}
	schedResp := doRequest(t, client, "POST", base+"/rooms/"+roomID+"/schedule/create", adminToken, schedBody)
	require.Equal(t, http.StatusCreated, schedResp.StatusCode)

	// 4. List available slots (the slot generator runs synchronously inside GenerateForSchedule)
	dateStr := tomorrow.Format("2006-01-02")
	slotResp := doRequest(t, client, "GET", base+"/rooms/"+roomID+"/slots/list?date="+dateStr, userToken, nil)
	require.Equal(t, http.StatusOK, slotResp.StatusCode)
	var slotOut struct {
		Slots []struct{ ID string `json:"id"` } `json:"slots"`
	}
	decodeJSON(t, slotResp, &slotOut)
	require.NotEmpty(t, slotOut.Slots, "should have available slots")

	// 5. Create a booking
	slotID := slotOut.Slots[0].ID
	bookBody := map[string]interface{}{"slotId": slotID}
	bookResp := doRequest(t, client, "POST", base+"/bookings/create", userToken, bookBody)
	require.Equal(t, http.StatusCreated, bookResp.StatusCode)
	var bookOut struct {
		Booking struct {
			ID     string `json:"id"`
			Status string `json:"status"`
			SlotID string `json:"slotId"`
		} `json:"booking"`
	}
	decodeJSON(t, bookResp, &bookOut)
	assert.Equal(t, "active", bookOut.Booking.Status)
	assert.Equal(t, slotID, bookOut.Booking.SlotID)
}

// ── Scenario 2: cancel booking ─────────────────────────────────────

func TestE2E_CancelBooking(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	client := srv.Client()
	base   := srv.URL

	adminToken := dummyLogin(t, client, base, "admin")
	userToken  := dummyLogin(t, client, base, "user")

	// Setup: room + schedule
	roomResp := doRequest(t, client, "POST", base+"/rooms/create", adminToken, map[string]interface{}{"name": "CancelRoom"})
	var roomOut struct {
		Room struct{ ID string `json:"id"` } `json:"room"`
	}
	decodeJSON(t, roomResp, &roomOut)
	roomID := roomOut.Room.ID

	doRequest(t, client, "POST", base+"/rooms/"+roomID+"/schedule/create", adminToken, map[string]interface{}{
		"daysOfWeek": []int{1, 2, 3, 4, 5, 6, 7},
		"startTime":  "08:00",
		"endTime":    "20:00",
	})

	// Get a slot
	dateStr := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	slotResp := doRequest(t, client, "GET", base+"/rooms/"+roomID+"/slots/list?date="+dateStr, userToken, nil)
	var slotOut struct {
		Slots []struct{ ID string `json:"id"` } `json:"slots"`
	}
	decodeJSON(t, slotResp, &slotOut)
	require.NotEmpty(t, slotOut.Slots)

	// Book
	bookResp := doRequest(t, client, "POST", base+"/bookings/create", userToken, map[string]interface{}{"slotId": slotOut.Slots[0].ID})
	var bookOut struct {
		Booking struct{ ID string `json:"id"` } `json:"booking"`
	}
	decodeJSON(t, bookResp, &bookOut)
	bookingID := bookOut.Booking.ID

	// Cancel
	cancelResp := doRequest(t, client, "POST", base+"/bookings/"+bookingID+"/cancel", userToken, nil)
	require.Equal(t, http.StatusOK, cancelResp.StatusCode)
	var cancelOut struct {
		Booking struct{ Status string `json:"status"` } `json:"booking"`
	}
	decodeJSON(t, cancelResp, &cancelOut)
	assert.Equal(t, "cancelled", cancelOut.Booking.Status)

	// Cancel again — idempotent
	cancelResp2 := doRequest(t, client, "POST", base+"/bookings/"+bookingID+"/cancel", userToken, nil)
	require.Equal(t, http.StatusOK, cancelResp2.StatusCode)
}

// ── Scenario 3: info endpoint ─────────────────────────────────────

func TestE2E_InfoEndpoint(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()
	resp, err := srv.Client().Get(srv.URL + "/_info")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ── Scenario 4: admin cannot create bookings ──────────────────────

func TestE2E_AdminCannotCreateBooking(t *testing.T) {
	srv, cleanup := newTestServer(t)
	defer cleanup()

	client := srv.Client()
	base   := srv.URL

	adminToken := dummyLogin(t, client, base, "admin")
	resp := doRequest(t, client, "POST", base+"/bookings/create", adminToken, map[string]interface{}{"slotId": "00000000-0000-0000-0000-000000000001"})
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

// ── Helpers ───────────────────────────────────────────────────────

func dummyLogin(t *testing.T, client *http.Client, base, role string) string {
	t.Helper()
	resp := doRequest(t, client, "POST", base+"/dummyLogin", "", map[string]interface{}{"role": role})
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var out struct{ Token string `json:"token"` }
	decodeJSON(t, resp, &out)
	require.NotEmpty(t, out.Token)
	return out.Token
}

func doRequest(t *testing.T, client *http.Client, method, url, token string, body interface{}) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req, err := http.NewRequest(method, url, &buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(v))
}
