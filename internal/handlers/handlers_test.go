package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/driver"
	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/models"
)

type postData struct {
	key   string
	value string
}

var theTests = []struct {
	name               string
	url                string
	method             string
	expectedStatusCode int
}{
	{"home", "/", "GET", http.StatusOK},
	{"about", "/about", "GET", http.StatusOK},
	{"gq", "/generals-quarters", "GET", http.StatusOK},
	{"ms", "/majors-suite", "GET", http.StatusOK},
	{"sa", "/search-availability", "GET", http.StatusOK},
	{"contact", "/contact", "GET", http.StatusOK},
	{"non-existent", "/green/eggs/and/ham", "GET", http.StatusNotFound},
	{"login", "/user/login", "GET", http.StatusOK},
	{"logout", "/user/logout", "GET", http.StatusOK},
	{"dashboard", "/admin/dashboard", "GET", http.StatusOK},
	{"new res", "/admin/reservations-new", "GET", http.StatusOK},
	{"all res", "/admin/reservations-all", "GET", http.StatusOK},
	{"show res", "/admin/reservations/new/1/show", "GET", http.StatusOK},
	{"show res cal", "/admin/reservations-calendar", "GET", http.StatusOK},
	{"show res cal with params", "/admin/reservations-calendar?y=2020&m=1", "GET", http.StatusOK},
}

// TestHandlers tests all routes that don't require extra tests (gets)
func TestHandlers(t *testing.T) {
	routes := getRoutes()
	ts := httptest.NewTLSServer(routes)
	defer ts.Close()

	for _, e := range theTests {
		resp, err := ts.Client().Get(ts.URL + e.url)
		if err != nil {
			t.Log(err)
			t.Fatal(err)
		}

		if resp.StatusCode != e.expectedStatusCode {
			t.Errorf("for %s, expected %d but got %d", e.name, e.expectedStatusCode, resp.StatusCode)
		}
	}
}

// data for the Reservation handler, /make-reservation route
var reservationTests = []struct {
	name               string
	reservation        models.Reservation
	expectedStatusCode int
	expectedLocation   string
	expectedHTML       string
}{
	{
		name: "reservation-in-session",
		reservation: models.Reservation{
			RoomID: 1,
			Room: models.Room{
				ID:       1,
				RoomName: "General's Quarters",
			},
		},
		expectedStatusCode: http.StatusOK,
		expectedHTML:       `action="/make-reservation"`,
	},
	{
		name:               "reservation-not-in-session",
		reservation:        models.Reservation{},
		expectedStatusCode: http.StatusSeeOther,
		expectedLocation:   "/",
		expectedHTML:       "",
	},
	{
		name: "non-existent-room",
		reservation: models.Reservation{
			RoomID: 100,
			Room: models.Room{
				ID:       100,
				RoomName: "General's Quarters",
			},
		},
		expectedStatusCode: http.StatusSeeOther,
		expectedLocation:   "/",
		expectedHTML:       "",
	},
}

// TestReservation tests the reservation handler
func TestReservation(t *testing.T) {
	for _, e := range reservationTests {
		req, _ := http.NewRequest("GET", "/make-reservation", nil)
		ctx := getCtx(req)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		if e.reservation.RoomID > 0 {
			session.Put(ctx, "reservation", e.reservation)
		}

		handler := http.HandlerFunc(Repo.GetMakeReservation)
		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedStatusCode {
			t.Errorf("%s returned wrong response code: got %d, wanted %d", e.name, rr.Code, e.expectedStatusCode)
		}

		if e.expectedLocation != "" {
			// get the URL from test
			actualLoc, _ := rr.Result().Location()
			if actualLoc.String() != e.expectedLocation {
				t.Errorf("failed %s: expected location %s, but got location %s", e.name, e.expectedLocation, actualLoc.String())
			}
		}

		if e.expectedHTML != "" {
			// read the response body into a string
			html := rr.Body.String()
			if !strings.Contains(html, e.expectedHTML) {
				t.Errorf("failed %s: expected to find %s but did not", e.name, e.expectedHTML)
			}
		}
	}
}

// postReservationTests is the test data for hte PostReservation handler test
var postReservationTests = []struct {
	name                 string
	postedData           url.Values
	expectedResponseCode int
	expectedLocation     string
	expectedHTML         string
}{
	{
		name: "valid-data",
		postedData: url.Values{
			"start_date": {"2050-01-01"},
			"end_date":   {"2050-01-02"},
			"first_name": {"John"},
			"last_name":  {"Smith"},
			"email":      {"john@smith.com"},
			"phone":      {"555-555-5555"},
			"room_id":    {"1"},
		},
		expectedResponseCode: http.StatusSeeOther,
		expectedHTML:         "",
		expectedLocation:     "/reservation-summary",
	},
	{
		name:                 "missing-post-body",
		postedData:           nil,
		expectedResponseCode: http.StatusSeeOther,
		expectedHTML:         "",
		expectedLocation:     "/",
	},
	{
		name: "invalid-start-date",
		postedData: url.Values{
			"start_date": {"invalid"},
			"end_date":   {"2050-01-02"},
			"first_name": {"John"},
			"last_name":  {"Smith"},
			"email":      {"john@smith.com"},
			"phone":      {"555-555-5555"},
			"room_id":    {"1"},
		},
		expectedResponseCode: http.StatusSeeOther,
		expectedHTML:         "",
		expectedLocation:     "/",
	},
	{
		name: "invalid-end-date",
		postedData: url.Values{
			"start_date": {"2050-01-01"},
			"end_date":   {"end"},
			"first_name": {"John"},
			"last_name":  {"Smith"},
			"email":      {"john@smith.com"},
			"phone":      {"555-555-5555"},
			"room_id":    {"1"},
		},
		expectedResponseCode: http.StatusSeeOther,
		expectedHTML:         "",
		expectedLocation:     "/",
	},
	{
		name: "invalid-room-id",
		postedData: url.Values{
			"start_date": {"2050-01-01"},
			"end_date":   {"2050-01-02"},
			"first_name": {"John"},
			"last_name":  {"Smith"},
			"email":      {"john@smith.com"},
			"phone":      {"555-555-5555"},
			"room_id":    {"invalid"},
		},
		expectedResponseCode: http.StatusSeeOther,
		expectedHTML:         "",
		expectedLocation:     "/",
	},
	{
		name: "invalid-data",
		postedData: url.Values{
			"start_date": {"2050-01-01"},
			"end_date":   {"2050-01-02"},
			"first_name": {"J"},
			"last_name":  {"Smith"},
			"email":      {"john@smith.com"},
			"phone":      {"555-555-5555"},
			"room_id":    {"1"},
		},
		expectedResponseCode: http.StatusOK,
		expectedHTML:         `action="/make-reservation"`,
		expectedLocation:     "",
	},
	{
		name: "database-insert-fails-reservation",
		postedData: url.Values{
			"start_date": {"2050-01-01"},
			"end_date":   {"2050-01-02"},
			"first_name": {"John"},
			"last_name":  {"Smith"},
			"email":      {"john@smith.com"},
			"phone":      {"555-555-5555"},
			"room_id":    {"2"},
		},
		expectedResponseCode: http.StatusSeeOther,
		expectedHTML:         "",
		expectedLocation:     "/",
	},
	{
		name: "database-insert-fails-restriction",
		postedData: url.Values{
			"start_date": {"2050-01-01"},
			"end_date":   {"2050-01-02"},
			"first_name": {"John"},
			"last_name":  {"Smith"},
			"email":      {"john@smith.com"},
			"phone":      {"555-555-5555"},
			"room_id":    {"1000"},
		},
		expectedResponseCode: http.StatusSeeOther,
		expectedHTML:         "",
		expectedLocation:     "/",
	},
}

// TestPostReservation tests the PostReservation handler
func TestPostReservation(t *testing.T) {
	for _, e := range postReservationTests {
		var req *http.Request
		if e.postedData != nil {
			req, _ = http.NewRequest("POST", "/make-reservation", strings.NewReader(e.postedData.Encode()))
		} else {
			req, _ = http.NewRequest("POST", "/make-reservation", nil)

		}
		ctx := getCtx(req)
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(Repo.PostMakeReservation)

		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedResponseCode {
			t.Errorf("%s returned wrong response code: got %d, wanted %d", e.name, rr.Code, e.expectedResponseCode)
		}

		if e.expectedLocation != "" {
			// get the URL from test
			actualLoc, _ := rr.Result().Location()
			if actualLoc.String() != e.expectedLocation {
				t.Errorf("failed %s: expected location %s, but got location %s", e.name, e.expectedLocation, actualLoc.String())
			}
		}

		if e.expectedHTML != "" {
			// read the response body into a string
			html := rr.Body.String()
			if !strings.Contains(html, e.expectedHTML) {
				t.Errorf("failed %s: expected to find %s but did not", e.name, e.expectedHTML)
			}
		}

	}
}

func TestNewRepo(t *testing.T) {
	var db driver.DB
	testRepo := NewRepo(&app, &db)

	if reflect.TypeOf(testRepo).String() != "*handlers.Repository" {
		t.Errorf("Did not get correct type from NewRepo: got %s, wanted *Repository", reflect.TypeOf(testRepo).String())
	}
}

// testAvailabilityJSONData is data for the AvailabilityJSON handler, /search-availability-json route
var testAvailabilityJSONData = []struct {
	name            string
	postedData      url.Values
	expectedOK      bool
	expectedMessage string
}{
	{
		name: "rooms not available",
		postedData: url.Values{
			"start":   {"2050-01-01"},
			"end":     {"2050-01-02"},
			"room_id": {"1"},
		},
		expectedOK: false,
	}, {
		name: "rooms are available",
		postedData: url.Values{
			"start":   {"2040-01-01"},
			"end":     {"2040-01-02"},
			"room_id": {"1"},
		},
		expectedOK: true,
	},
	{
		name:            "empty post body",
		postedData:      nil,
		expectedOK:      false,
		expectedMessage: "Internal Server Error",
	},
	{
		name: "database query fails",
		postedData: url.Values{
			"start":   {"2060-01-01"},
			"end":     {"2060-01-02"},
			"room_id": {"1"},
		},
		expectedOK:      false,
		expectedMessage: "Error querying database",
	},
}

// TestAvailabilityJSON tests the AvailabilityJSON handler
func TestAvailabilityJSON(t *testing.T) {
	for _, e := range testAvailabilityJSONData {
		// create request, get the context with session, set header, create recorder
		var req *http.Request
		if e.postedData != nil {
			req, _ = http.NewRequest("POST", "/search-availability-json", strings.NewReader(e.postedData.Encode()))
		} else {
			req, _ = http.NewRequest("POST", "/search-availability-json", nil)
		}
		ctx := getCtx(req)
		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		// make our handler a http.HandlerFunc and call
		handler := http.HandlerFunc(Repo.SearchAvailabilityJSON)
		handler.ServeHTTP(rr, req)

		var j jsonResponse
		err := json.Unmarshal([]byte(rr.Body.String()), &j)
		if err != nil {
			t.Error("failed to parse json!")
		}

		if j.OK != e.expectedOK {
			t.Errorf("%s: expected %v but got %v", e.name, e.expectedOK, j.OK)
		}
	}
}

// testPostAvailabilityData is data for the PostAvailability handler test, /search-availability
var testPostAvailabilityData = []struct {
	name               string
	postedData         url.Values
	expectedStatusCode int
	expectedLocation   string
}{
	{
		name: "rooms not available",
		postedData: url.Values{
			"start": {"2050-01-01"},
			"end":   {"2050-01-02"},
		},
		expectedStatusCode: http.StatusSeeOther,
	},
	{
		name: "rooms are available",
		postedData: url.Values{
			"start":   {"2040-01-01"},
			"end":     {"2040-01-02"},
			"room_id": {"1"},
		},
		expectedStatusCode: http.StatusOK,
	},
	{
		name:               "empty post body",
		postedData:         url.Values{},
		expectedStatusCode: http.StatusSeeOther,
	},
	{
		name: "start date wrong format",
		postedData: url.Values{
			"start":   {"invalid"},
			"end":     {"2040-01-02"},
			"room_id": {"1"},
		},
		expectedStatusCode: http.StatusSeeOther,
	},
	{
		name: "end date wrong format",
		postedData: url.Values{
			"start": {"2040-01-01"},
			"end":   {"invalid"},
		},
		expectedStatusCode: http.StatusSeeOther,
	},
	{
		name: "database query fails",
		postedData: url.Values{
			"start": {"2060-01-01"},
			"end":   {"2060-01-02"},
		},
		expectedStatusCode: http.StatusSeeOther,
	},
}

// TestPostAvailability tests the PostAvailabilityHandler
func TestPostAvailability(t *testing.T) {
	for _, e := range testPostAvailabilityData {
		req, _ := http.NewRequest("POST", "/search-availability", strings.NewReader(e.postedData.Encode()))

		// get the context with session
		ctx := getCtx(req)
		req = req.WithContext(ctx)

		// set the request header
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		// make our handler a http.HandlerFunc and call
		handler := http.HandlerFunc(Repo.PostSearchAvailability)
		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedStatusCode {
			t.Errorf("%s gave wrong status code: got %d, wanted %d", e.name, rr.Code, e.expectedStatusCode)
		}
	}
}

// reservationSummaryTests is the data to test ReservationSummary handler
var reservationSummaryTests = []struct {
	name               string
	reservation        models.Reservation
	url                string
	expectedStatusCode int
	expectedLocation   string
}{
	{
		name: "res-in-session",
		reservation: models.Reservation{
			RoomID: 1,
			Room: models.Room{
				ID:       1,
				RoomName: "General's Quarters",
			},
		},
		url:                "/reservation-summary",
		expectedStatusCode: http.StatusOK,
		expectedLocation:   "",
	},
	{
		name:               "res-not-in-session",
		reservation:        models.Reservation{},
		url:                "/reservation-summary",
		expectedStatusCode: http.StatusSeeOther,
		expectedLocation:   "/",
	},
}

// TestReservationSummary tests the ReservationSummaryHandler
func TestReservationSummary(t *testing.T) {
	for _, e := range reservationSummaryTests {
		req, _ := http.NewRequest("GET", e.url, nil)
		ctx := getCtx(req)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		if e.reservation.RoomID > 0 {
			session.Put(ctx, "reservation", e.reservation)
		}

		handler := http.HandlerFunc(Repo.ReservationSummary)

		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedStatusCode {
			t.Errorf("%s returned wrong response code: got %d, wanted %d", e.name, rr.Code, e.expectedStatusCode)
		}

		if e.expectedLocation != "" {
			actualLoc, _ := rr.Result().Location()
			if actualLoc.String() != e.expectedLocation {
				t.Errorf("failed %s: expected location %s, but got location %s", e.name, e.expectedLocation, actualLoc.String())
			}
		}
	}
}

// chooseRoomTests is the data for ChooseRoom handler tests, /choose-room/{id}
var chooseRoomTests = []struct {
	name               string
	reservation        models.Reservation
	url                string
	expectedStatusCode int
	expectedLocation   string
}{
	{
		name: "reservation-in-session",
		reservation: models.Reservation{
			RoomID: 1,
			Room: models.Room{
				ID:       1,
				RoomName: "General's Quarters",
			},
		},
		url:                "/choose-room/1",
		expectedStatusCode: http.StatusSeeOther,
		expectedLocation:   "/make-reservation",
	},
	{
		name:               "reservation-not-in-session",
		reservation:        models.Reservation{},
		url:                "/choose-room/1",
		expectedStatusCode: http.StatusSeeOther,
		expectedLocation:   "/",
	},
	{
		name:               "malformed-url",
		reservation:        models.Reservation{},
		url:                "/choose-room/fish",
		expectedStatusCode: http.StatusSeeOther,
		expectedLocation:   "/",
	},
}

// TestChooseRoom tests the ChooseRoom handler
func TestChooseRoom(t *testing.T) {
	for _, e := range chooseRoomTests {
		req, _ := http.NewRequest("GET", e.url, nil)
		ctx := getCtx(req)
		req = req.WithContext(ctx)
		// set the RequestURI on the request so that we can grab the ID from the URL
		req.RequestURI = e.url

		rr := httptest.NewRecorder()
		if e.reservation.RoomID > 0 {
			session.Put(ctx, "reservation", e.reservation)
		}

		handler := http.HandlerFunc(Repo.ChooseRoom)
		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedStatusCode {
			t.Errorf("%s returned wrong response code: got %d, wanted %d", e.name, rr.Code, e.expectedStatusCode)
		}

		if e.expectedLocation != "" {
			actualLoc, _ := rr.Result().Location()
			if actualLoc.String() != e.expectedLocation {
				t.Errorf("failed %s: expected location %s, but got location %s", e.name, e.expectedLocation, actualLoc.String())
			}
		}
	}
}

// bookRoomTests is the data for the BookRoom handler tests
var bookRoomTests = []struct {
	name               string
	url                string
	expectedStatusCode int
}{
	{
		name:               "database-works",
		url:                "/book-room?s=2050-01-01&e=2050-01-02&id=1",
		expectedStatusCode: http.StatusSeeOther,
	},
	{
		name:               "database-fails",
		url:                "/book-room?s=2040-01-01&e=2040-01-02&id=4",
		expectedStatusCode: http.StatusSeeOther,
	},
}

// TestBookRoom tests the BookRoom handler
func TestBookRoom(t *testing.T) {
	reservation := models.Reservation{
		RoomID: 1,
		Room: models.Room{
			ID:       1,
			RoomName: "General's Quarters",
		},
	}

	for _, e := range bookRoomTests {
		req, _ := http.NewRequest("GET", e.url, nil)
		ctx := getCtx(req)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		session.Put(ctx, "reservation", reservation)

		handler := http.HandlerFunc(Repo.BookRoom)

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Errorf("%s failed: returned wrong response code: got %d, wanted %d", e.name, rr.Code, e.expectedStatusCode)
		}
	}
}

// loginTests is the data for the Login handler tests
var loginTests = []struct {
	name               string
	email              string
	expectedStatusCode int
	expectedHTML       string
	expectedLocation   string
}{
	{
		"valid-credentials",
		"me@here.ca",
		http.StatusSeeOther,
		"",
		"/",
	},
	{
		"invalid-credentials",
		"jack@nimble.com",
		http.StatusSeeOther,
		"",
		"/user/login",
	},
	{
		"invalid-data",
		"j",
		http.StatusOK,
		`action="/user/login"`,
		"",
	},
}

func TestLogin(t *testing.T) {
	// range through all tests
	for _, e := range loginTests {
		postedData := url.Values{}
		postedData.Add("email", e.email)
		postedData.Add("password", "password")

		// create request
		req, _ := http.NewRequest("POST", "/user/login", strings.NewReader(postedData.Encode()))
		ctx := getCtx(req)
		req = req.WithContext(ctx)

		// set the header
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		// call the handler
		handler := http.HandlerFunc(Repo.PostShowLogin)
		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedStatusCode {
			t.Errorf("failed %s: expected code %d, but got %d", e.name, e.expectedStatusCode, rr.Code)
		}

		if e.expectedLocation != "" {
			// get the URL from test
			actualLoc, _ := rr.Result().Location()
			if actualLoc.String() != e.expectedLocation {
				t.Errorf("failed %s: expected location %s, but got location %s", e.name, e.expectedLocation, actualLoc.String())
			}
		}

		// checking for expected values in HTML
		if e.expectedHTML != "" {
			// read the response body into a string
			html := rr.Body.String()
			if !strings.Contains(html, e.expectedHTML) {
				t.Errorf("failed %s: expected to find %s but did not", e.name, e.expectedHTML)
			}
		}
	}
}

var adminPostShowReservationTests = []struct {
	name                 string
	url                  string
	postedData           url.Values
	expectedResponseCode int
	expectedLocation     string
	expectedHTML         string
}{
	{
		name: "valid-data-from-new",
		url:  "/admin/reservations/new/1/show",
		postedData: url.Values{
			"first_name": {"John"},
			"last_name":  {"Smith"},
			"email":      {"john@smith.com"},
			"phone":      {"555-555-5555"},
		},
		expectedResponseCode: http.StatusSeeOther,
		expectedLocation:     "/admin/reservations-new",
		expectedHTML:         "",
	},
	{
		name: "valid-data-from-all",
		url:  "/admin/reservations/all/1/show",
		postedData: url.Values{
			"first_name": {"John"},
			"last_name":  {"Smith"},
			"email":      {"john@smith.com"},
			"phone":      {"555-555-5555"},
		},
		expectedResponseCode: http.StatusSeeOther,
		expectedLocation:     "/admin/reservations-all",
		expectedHTML:         "",
	},
	{
		name: "valid-data-from-cal",
		url:  "/admin/reservations/cal/1/show",
		postedData: url.Values{
			"first_name": {"John"},
			"last_name":  {"Smith"},
			"email":      {"john@smith.com"},
			"phone":      {"555-555-5555"},
			"year":       {"2022"},
			"month":      {"01"},
		},
		expectedResponseCode: http.StatusSeeOther,
		expectedLocation:     "/admin/reservations-calendar?y=2022&m=01",
		expectedHTML:         "",
	},
}

// TestAdminPostShowReservation tests the AdminPostReservation handler
func TestAdminPostShowReservation(t *testing.T) {
	for _, e := range adminPostShowReservationTests {
		var req *http.Request
		if e.postedData != nil {
			req, _ = http.NewRequest("POST", "/user/login", strings.NewReader(e.postedData.Encode()))
		} else {
			req, _ = http.NewRequest("POST", "/user/login", nil)
		}
		ctx := getCtx(req)
		req = req.WithContext(ctx)
		req.RequestURI = e.url

		// set the header
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		// call the handler
		handler := http.HandlerFunc(Repo.AdminPostShowReservation)
		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedResponseCode {
			t.Errorf("failed %s: expected code %d, but got %d", e.name, e.expectedResponseCode, rr.Code)
		}

		if e.expectedLocation != "" {
			// get the URL from test
			actualLoc, _ := rr.Result().Location()
			if actualLoc.String() != e.expectedLocation {
				t.Errorf("failed %s: expected location %s, but got location %s", e.name, e.expectedLocation, actualLoc.String())
			}
		}

		// checking for expected values in HTML
		if e.expectedHTML != "" {
			// read the response body into a string
			html := rr.Body.String()
			if !strings.Contains(html, e.expectedHTML) {
				t.Errorf("failed %s: expected to find %s but did not", e.name, e.expectedHTML)
			}
		}
	}
}

var adminPostReservationCalendarTests = []struct {
	name                 string
	postedData           url.Values
	expectedResponseCode int
	expectedLocation     string
	expectedHTML         string
	blocks               int
	reservations         int
}{
	{
		name: "cal",
		postedData: url.Values{
			"year":  {time.Now().Format("2006")},
			"month": {time.Now().Format("01")},
			fmt.Sprintf("add_block_1_%s", time.Now().AddDate(0, 0, 2).Format("2006-01-2")): {"1"},
		},
		expectedResponseCode: http.StatusSeeOther,
	},
	{
		name:                 "cal-blocks",
		postedData:           url.Values{},
		expectedResponseCode: http.StatusSeeOther,
		blocks:               1,
	},
	{
		name:                 "cal-res",
		postedData:           url.Values{},
		expectedResponseCode: http.StatusSeeOther,
		reservations:         1,
	},
}

func TestPostReservationCalendar(t *testing.T) {
	for _, e := range adminPostReservationCalendarTests {
		var req *http.Request
		if e.postedData != nil {
			req, _ = http.NewRequest("POST", "/admin/reservations-calendar", strings.NewReader(e.postedData.Encode()))
		} else {
			req, _ = http.NewRequest("POST", "/admin/reservations-calendar", nil)
		}
		ctx := getCtx(req)
		req = req.WithContext(ctx)

		now := time.Now()
		bm := make(map[string]int)
		rm := make(map[string]int)

		currentYear, currentMonth, _ := now.Date()
		currentLocation := now.Location()

		firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)
		lastOfMonth := firstOfMonth.AddDate(0, 1, -1)

		for d := firstOfMonth; d.After(lastOfMonth) == false; d = d.AddDate(0, 0, 1) {
			rm[d.Format("2006-01-2")] = 0
			bm[d.Format("2006-01-2")] = 0
		}

		if e.blocks > 0 {
			bm[firstOfMonth.Format("2006-01-2")] = e.blocks
		}

		if e.reservations > 0 {
			rm[lastOfMonth.Format("2006-01-2")] = e.reservations
		}

		session.Put(ctx, "block_map_1", bm)
		session.Put(ctx, "reservation_map_1", rm)

		// set the header
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()

		// call the handler
		handler := http.HandlerFunc(Repo.AdminPostReservationsCalendar)
		handler.ServeHTTP(rr, req)

		if rr.Code != e.expectedResponseCode {
			t.Errorf("failed %s: expected code %d, but got %d", e.name, e.expectedResponseCode, rr.Code)
		}

	}
}

var adminProcessReservationTests = []struct {
	name                 string
	queryParams          string
	expectedResponseCode int
	expectedLocation     string
}{
	{
		name:                 "process-reservation",
		queryParams:          "",
		expectedResponseCode: http.StatusSeeOther,
		expectedLocation:     "",
	},
	{
		name:                 "process-reservation-back-to-cal",
		queryParams:          "?y=2021&m=12",
		expectedResponseCode: http.StatusSeeOther,
		expectedLocation:     "",
	},
}

func TestAdminProcessReservation(t *testing.T) {
	for _, e := range adminProcessReservationTests {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/process-reservation/cal/1/do%s", e.queryParams), nil)
		ctx := getCtx(req)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(Repo.AdminProcessReservation)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Errorf("failed %s: expected code %d, but got %d", e.name, e.expectedResponseCode, rr.Code)
		}
	}
}

var adminDeleteReservationTests = []struct {
	name                 string
	queryParams          string
	expectedResponseCode int
	expectedLocation     string
}{
	{
		name:                 "delete-reservation",
		queryParams:          "",
		expectedResponseCode: http.StatusSeeOther,
		expectedLocation:     "",
	},
	{
		name:                 "delete-reservation-back-to-cal",
		queryParams:          "?y=2021&m=12",
		expectedResponseCode: http.StatusSeeOther,
		expectedLocation:     "",
	},
}

func TestAdminDeleteReservation(t *testing.T) {
	for _, e := range adminDeleteReservationTests {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/admin/process-reservation/cal/1/do%s", e.queryParams), nil)
		ctx := getCtx(req)
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		handler := http.HandlerFunc(Repo.AdminDeleteReservation)
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Errorf("failed %s: expected code %d, but got %d", e.name, e.expectedResponseCode, rr.Code)
		}
	}
}

// gets the context
func getCtx(req *http.Request) context.Context {
	ctx, err := session.Load(req.Context(), req.Header.Get("X-Session"))
	if err != nil {
		log.Println(err)
	}
	return ctx
}

// Testing Version #1

// package handlers

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"net/http/httptest"
// 	"net/url"
// 	"strings"
// 	"testing"
// 	"time"

// 	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/models"
// )

// type postData struct {
// 	key   string
// 	value string
// }

// // theTests - test application routes
// // params name - name of test func
// // params url -  the path which matched by our routes
// // params method - GET POST
// // params expetedStatusCode - set base on https statuscode 400,200 etc...
// var theTests = []struct {
// 	name              string
// 	url               string
// 	method            string
// 	expetedStatusCode int
// }{
// 	{"home", "/", "GET", http.StatusOK},
// 	{"about", "/about", "GET", http.StatusOK},
// 	{"gqg", "/rooms/general-quarters", "GET", http.StatusOK},
// 	{"mqg", "/rooms/major-quarters", "GET", http.StatusOK},
// 	{"sag", "/search-availability", "GET", http.StatusOK},
// 	// {"sap", "/search-availability", "POST", []postData{
// 	// 	{key: "start", value: "2023-11-24"},
// 	// 	{key: "start", value: "2023-11-29"},
// 	// }, http.StatusOK},
// 	// {"saj", "/search-availability-json", "POST", []postData{
// 	// 	{key: "start", value: "2023-11-24"},
// 	// 	{key: "start", value: "2023-11-29"},
// 	// }, http.StatusOK},
// 	// {"mrg", "/make-reservation", "GET", []postData{}, http.StatusOK},
// 	// {"mrp", "/make-reservation", "POST", []postData{
// 	// 	{key: "first_name", value: "Jericho"},
// 	// 	{key: "last_name", value: "Baisa"},
// 	// 	{key: "email", value: "jbgmail.com"},
// 	// 	{key: "phone", value: "555-555-5555"},
// 	// }, http.StatusOK},
// 	// {"rs", "/reservation-summary", "GET", []postData{}, http.StatusOK},
// 	{"contact", "/contact", "GET", http.StatusOK},

// 	{"login", "/user/login", "GET", http.StatusOK},
// 	{"logout", "/user/logout", "GET", http.StatusOK},
// 	{"dashboard", "/admin/dashboard", "GET", http.StatusOK},
// 	{"new reservation", "/admin/reservations-new", "GET", http.StatusOK},
// 	{"all reservation", "/admin/reservations-all", "GET", http.StatusOK},
// 	{"show reservation", "/admin/reservations/new/1/show", "GET", http.StatusOK},
// }

// // how are we going to create a webserver that actually returns a status code, something we can post to, in effect create a sever
// // and we also need to create a client that can call that server
// func TestHandlers(t *testing.T) {
// 	routes := getRoutes()
// 	ts := httptest.NewTLSServer(routes)
// 	defer ts.Close() // it is a best practice to close the server

// 	// we can make a post and get request test here
// 	for _, e := range theTests {
// 		if e.method == "GET" {
// 			resp, err := ts.Client().Get(ts.URL + e.url)
// 			if err != nil {
// 				t.Log(err)
// 				t.Fatal(err)
// 			}

// 			if resp.StatusCode != e.expetedStatusCode {
// 				t.Errorf("for %s expected %d  but got %d", e.name, e.expetedStatusCode, resp.StatusCode)
// 			}
// 		}
// 		// else {
// 		// 	// post
// 		// 	values := url.Values{}
// 		// 	for _, x := range e.params {
// 		// 		values.Add(x.key, x.value)

// 		// 	}

// 		// 	resp, err := ts.Client().PostForm(ts.URL+e.url, values)
// 		// 	if err != nil {
// 		// 		t.Log(err)
// 		// 		t.Fatal(err)
// 		// 	}

// 		// 	if resp.StatusCode != e.expetedStatusCode {
// 		// 		t.Errorf("for %s expected %d  but got %d", e.name, e.expetedStatusCode, resp.StatusCode)
// 		// 	}
// 		// }
// 	}

// }

// func TestRepository_GetMakeReservation(t *testing.T) {
// 	reservation := models.Reservation{
// 		FirstName: "Jericho",
// 		LastName:  "Baisa",
// 		Email:     "jb@gmail.com",

// 		RoomID: 1,
// 		Room: models.Room{
// 			ID:       1,
// 			RoomName: "General Quarters",
// 		},
// 	}

// 	req, _ := http.NewRequest("GET", "/make-reservation", nil)
// 	ctx := getCtx(req)
// 	req = req.WithContext(ctx)

// 	//request recorder - is basically simulating what we can get from the request response cycle when someone fires up a web browser, hits our website
// 	//gets to a handler, passes it a request, gets a response writer and the response writer writes the response to the web browser.
// 	// This fakes that entire process or the part of it that we need for our recorder
// 	rr := httptest.NewRecorder()
// 	session.Put(ctx, "reservation", reservation)
// 	handler := http.HandlerFunc(Repo.GetMakeReservation)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusOK {
// 		t.Errorf("GetMakeReservation handler returned wrong response code: got %d, wanted %d", rr.Code, http.StatusOK)
// 	}

// 	// test case where reservation is not in session (reset everything)
// 	req, _ = http.NewRequest("GET", "/make-reservation", nil)
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	rr = httptest.NewRecorder()
// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("GetMakeReservation handler returned wrong response code: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	//test with non-existent room
// 	req, _ = http.NewRequest("GET", "/make-reservation", nil)
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	rr = httptest.NewRecorder()
// 	reservation.RoomID = 100
// 	session.Put(ctx, "reservation", reservation)
// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("GetMakeReservation handler returned wrong response code: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// }

// func TestRepository_PostMakeReservation(t *testing.T) {
// 	layout := "2006-01-02"
// 	start := "2050-11-22"
// 	end := "2050-11-23"

// 	startDate, err := time.Parse(layout, start)
// 	if err != nil {
// 		t.Errorf("PostMakeReservation handler returned error parsing start date: got %s, wanted %s", err, start)
// 	}

// 	endDate, err := time.Parse(layout, end)
// 	if err != nil {
// 		t.Errorf("PostMakeReservation handler returned error parsing end date: got %s, wanted %s", err, end)
// 	}

// 	reservation := models.Reservation{
// 		FirstName: "Jericho",
// 		LastName:  "Baisa",
// 		Email:     "jb@gmail.com",
// 		Phone:     "555-555-555",
// 		StartDate: startDate,
// 		EndDate:   endDate,

// 		RoomID: 1,
// 		Room: models.Room{
// 			ID:       1,
// 			RoomName: "General Quarters",
// 		},
// 	}

// 	// reqBody := "start_date=2050-11-22"
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "end_date=2050-11-23")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "first_name=Echo")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "last_name=Lucy")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "email=le@gmail.com")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "phone=0987654321")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "room_id=1")

// 	postedData := url.Values{}
// 	postedData.Add("start_date", "2050-11-22")
// 	postedData.Add("end_date", "2050-11-23")
// 	postedData.Add("first_name", "Echo")
// 	postedData.Add("last_name", "Lucy")
// 	postedData.Add("email", "el@gmail.com")
// 	postedData.Add("phone", "0987654321")
// 	postedData.Add("room_id", "1")

// 	req, _ := http.NewRequest("POST", "/make-reservation", strings.NewReader(postedData.Encode()))
// 	ctx := getCtx(req)
// 	session.Put(ctx, "reservation", reservation)
// 	req = req.WithContext(ctx)

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	rr := httptest.NewRecorder()

// 	handler := http.HandlerFunc(Repo.PostMakeReservation)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusSeeOther {
// 		t.Errorf("PostMakeReservation handler returned wrong response code: got %d, wanted %d", rr.Code, http.StatusSeeOther)
// 	}

// 	//test post reservation not in session
// 	req, _ = http.NewRequest("POST", "/make-reservation", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
// 	rr = httptest.NewRecorder()

// 	handler = http.HandlerFunc(Repo.PostMakeReservation)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostMakeReservation handler returned wrong response code for missing session data: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	// test for invalid form
// 	// reqBody = "start_date=2050-11-22"
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "end_date=2050-11-23")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "first_name=EB")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "last_name=Lucy")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "email=le@gmail.com")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "phone=0987654321")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "room_id=1")

// 	postedData = url.Values{}
// 	postedData.Add("start_date", "2050-11-22")
// 	postedData.Add("end_date", "2050-11-23")
// 	postedData.Add("first_name", "EB")
// 	postedData.Add("last_name", "Lucy")
// 	postedData.Add("email", "el@gmail.com")
// 	postedData.Add("phone", "0987654321")
// 	postedData.Add("room_id", "1")

// 	req, _ = http.NewRequest("POST", "/make-reservation", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	session.Put(ctx, "reservation", reservation)
// 	req = req.WithContext(ctx)
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
// 	rr = httptest.NewRecorder()

// 	handler = http.HandlerFunc(Repo.PostMakeReservation)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostMakeReservation handler returned wrong response code for invalid form data: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	// test for invalid reservation
// 	reservation = models.Reservation{
// 		ID:        2,
// 		FirstName: "Jericho",
// 		LastName:  "Baisa",
// 		Email:     "jb@gmail.com",
// 		Phone:     "555-555-555",
// 		StartDate: startDate,
// 		EndDate:   endDate,

// 		RoomID: 2,
// 		Room: models.Room{
// 			ID:       1,
// 			RoomName: "General Quarters",
// 		},
// 	}
// 	// reqBody = "start_date=2050-11-22"
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "end_date=2050-11-23")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "first_name=Jericho")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "last_name=Lucy")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "email=le@gmail.com")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "phone=0987654321")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "room_id=1")

// 	postedData = url.Values{}
// 	postedData.Add("start_date", "2050-11-22")
// 	postedData.Add("end_date", "2050-11-23")
// 	postedData.Add("first_name", "Echo")
// 	postedData.Add("last_name", "Lucy")
// 	postedData.Add("email", "el@gmail.com")
// 	postedData.Add("phone", "0987654321")
// 	postedData.Add("room_id", "1")

// 	req, _ = http.NewRequest("POST", "/make-reservation", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	session.Put(ctx, "reservation", reservation)
// 	req = req.WithContext(ctx)
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
// 	rr = httptest.NewRecorder()

// 	handler = http.HandlerFunc(Repo.PostMakeReservation)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostMakeReservation handler failed when trying to fail inserting reservation data: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	// test for invalid room restriction
// 	reservation = models.Reservation{
// 		ID:        1,
// 		FirstName: "Jericho",
// 		LastName:  "Baisa",
// 		Email:     "jb@gmail.com",
// 		Phone:     "555-555-555",
// 		StartDate: startDate,
// 		EndDate:   endDate,

// 		RoomID: 22,
// 		Room: models.Room{
// 			ID:       1,
// 			RoomName: "General Quarters",
// 		},
// 	}

// 	// reqBody = "start_date=2050-11-22"
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "end_date=2050-11-23")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "first_name=Jericho")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "last_name=Lucy")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "email=le@gmail.com")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "phone=0987654321")
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "room_id=1")
// 	postedData = url.Values{}
// 	postedData.Add("start_date", "2050-11-22")
// 	postedData.Add("end_date", "2050-11-23")
// 	postedData.Add("first_name", "Echo")
// 	postedData.Add("last_name", "Lucy")
// 	postedData.Add("email", "el@gmail.com")
// 	postedData.Add("phone", "0987654321")
// 	postedData.Add("room_id", "1")

// 	req, _ = http.NewRequest("POST", "/make-reservation", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	session.Put(ctx, "reservation", reservation)
// 	req = req.WithContext(ctx)
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
// 	rr = httptest.NewRecorder()

// 	handler = http.HandlerFunc(Repo.PostMakeReservation)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostMakeReservation handler failed when trying to fail inserting reservation data: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	// test for missing body
// 	req, _ = http.NewRequest("POST", "/make-reservation", nil)
// 	ctx = getCtx(req)
// 	session.Put(ctx, "reservation", reservation)
// 	req = req.WithContext(ctx)
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
// 	rr = httptest.NewRecorder()

// 	handler = http.HandlerFunc(Repo.PostMakeReservation)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostMakeReservation handler returned wrong response code for missing post body: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}
// }

// func TestRepository_ReservationSummary(t *testing.T) {
// 	layout := "2006-01-02"
// 	start := "2050-11-22"
// 	end := "2050-11-23"

// 	startDate, err := time.Parse(layout, start)
// 	if err != nil {
// 		t.Errorf("ReservationSummary handler returned error parsing start date: got %s, wanted %s", err, start)
// 	}

// 	endDate, err := time.Parse(layout, end)
// 	if err != nil {
// 		t.Errorf("ReservationSummary handler returned error parsing end date: got %s, wanted %s", err, end)
// 	}
// 	// test for with session data
// 	reservation := models.Reservation{
// 		StartDate: startDate,
// 		EndDate:   endDate,
// 		RoomID:    1,
// 		Room: models.Room{
// 			ID:       1,
// 			RoomName: "General Quarters",
// 		},
// 	}
// 	req, _ := http.NewRequest("GET", "/reservation-summary", nil)
// 	ctx := getCtx(req)
// 	session.Put(ctx, "reservation", reservation)
// 	req = req.WithContext(ctx)
// 	rr := httptest.NewRecorder()
// 	handler := http.HandlerFunc(Repo.ReservationSummary)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusOK {
// 		t.Errorf("ReservationSummary handler failed when trying to fail with session data: got %d, wanted %d", rr.Code, http.StatusOK)
// 	}

// 	// test failure session data
// 	req, _ = http.NewRequest("GET", "/reservation-summary", nil)
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	rr = httptest.NewRecorder()

// 	handler = http.HandlerFunc(Repo.ReservationSummary)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("ReservationSummary handler failed when trying to fail no session: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}
// }

// func TestRepository_SearchAvailabilityJSON(t *testing.T) {
// 	reqBody := "start=2050-01-01"
// 	reqBody = fmt.Sprintf("%s&%s", reqBody, "end=2050-01-02")
// 	reqBody = fmt.Sprintf("%s&%s", reqBody, "room_id=1")

// 	//create request
// 	req, _ := http.NewRequest("POST", "/search-availability", strings.NewReader(reqBody))

// 	// get context with session
// 	ctx := getCtx(req)
// 	req = req.WithContext(ctx)

// 	// set the request header
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	// get response recorder
// 	rr := httptest.NewRecorder()

// 	// make handler handlerfunc
// 	handler := http.HandlerFunc(Repo.SearchAvailabilityJSON)

// 	// make request to our handler
// 	handler.ServeHTTP(rr, req)

// 	var j jsonRespone
// 	err := json.Unmarshal(rr.Body.Bytes(), &j)
// 	if err != nil {
// 		t.Error("failed to parse json")
// 	}

// 	// test to failure parsing a form
// 	//create request
// 	req, _ = http.NewRequest("POST", "/search-availability", nil)

// 	// get context with session
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)

// 	// set the request header
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	// get response recorder
// 	rr = httptest.NewRecorder()

// 	// make handler handlerfunc
// 	handler = http.HandlerFunc(Repo.SearchAvailabilityJSON)

// 	// make request to our handler
// 	handler.ServeHTTP(rr, req)

// 	err = json.Unmarshal(rr.Body.Bytes(), &j)
// 	if err != nil {
// 		t.Error("failed to parse json")
// 	}

// 	// since we specified a start date > 2049-12-31, we expect no availability
// 	if j.OK {
// 		t.Error("Got availability when none was expected in AvailabilityJSON")
// 	}

// 	// test to failure start date
// 	reqBody = "start=invalid"
// 	reqBody = fmt.Sprintf("%s&%s", reqBody, "end=2050-01-02")
// 	reqBody = fmt.Sprintf("%s&%s", reqBody, "room_id=1")

// 	//create request
// 	req, _ = http.NewRequest("POST", "/search-availability", strings.NewReader(reqBody))

// 	// get context with session
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)

// 	// set the request header
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	// get response recorder
// 	rr = httptest.NewRecorder()

// 	// make handler handlerfunc
// 	handler = http.HandlerFunc(Repo.SearchAvailabilityJSON)

// 	// make request to our handler
// 	handler.ServeHTTP(rr, req)

// 	err = json.Unmarshal(rr.Body.Bytes(), &j)
// 	if err != nil {
// 		t.Error("failed to parse json")
// 	}

// 	// since we specified a start date is invalid
// 	if j.OK {
// 		t.Error("Got availability when some was expected in AvailabilityJSON")
// 	}

// 	// test to  end date
// 	reqBody = "start=2040-01-01"
// 	reqBody = fmt.Sprintf("%s&%s", reqBody, "end=invalid")
// 	reqBody = fmt.Sprintf("%s&%s", reqBody, "room_id=1")

// 	//create request
// 	req, _ = http.NewRequest("POST", "/search-availability", strings.NewReader(reqBody))

// 	// get context with session
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)

// 	// set the request header
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	// get response recorder
// 	rr = httptest.NewRecorder()

// 	// make handler handlerfunc
// 	handler = http.HandlerFunc(Repo.SearchAvailabilityJSON)

// 	// make request to our handler
// 	handler.ServeHTTP(rr, req)

// 	err = json.Unmarshal(rr.Body.Bytes(), &j)
// 	if err != nil {
// 		t.Error("failed to parse json")
// 	}

// 	// since we specified a end date isinvalid
// 	if j.OK {
// 		t.Error("Got availability when some was expected in AvailabilityJSON")
// 	}

// 	// test to  room id
// 	reqBody = "start=2040-01-01"
// 	reqBody = fmt.Sprintf("%s&%s", reqBody, "end=2040-01-02")
// 	reqBody = fmt.Sprintf("%s&%s", reqBody, "room_id=invalid")

// 	//create request
// 	req, _ = http.NewRequest("POST", "/search-availability", strings.NewReader(reqBody))

// 	// get context with session
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)

// 	// set the request header
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	// get response recorder
// 	rr = httptest.NewRecorder()

// 	// make handler handlerfunc
// 	handler = http.HandlerFunc(Repo.SearchAvailabilityJSON)

// 	// make request to our handler
// 	handler.ServeHTTP(rr, req)

// 	err = json.Unmarshal(rr.Body.Bytes(), &j)
// 	if err != nil {
// 		t.Error("failed to parse json")
// 	}

// 	// since we specified a room id is invalid
// 	if j.OK {
// 		t.Error("Got availability when some was expected in AvailabilityJSON")
// 	}

// 	// test to  room id not exist
// 	reqBody = "start=2040-01-01"
// 	reqBody = fmt.Sprintf("%s&%s", reqBody, "end=2040-01-02")
// 	reqBody = fmt.Sprintf("%s&%s", reqBody, "room_id=22")

// 	//create request
// 	req, _ = http.NewRequest("POST", "/search-availability", strings.NewReader(reqBody))

// 	// get context with session
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)

// 	// set the request header
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	// get response recorder
// 	rr = httptest.NewRecorder()

// 	// make handler handlerfunc
// 	handler = http.HandlerFunc(Repo.SearchAvailabilityJSON)

// 	// make request to our handler
// 	handler.ServeHTTP(rr, req)

// 	err = json.Unmarshal(rr.Body.Bytes(), &j)
// 	if err != nil {
// 		t.Error("failed to parse json")
// 	}

// 	// since we specified a room id not exist
// 	if j.OK {
// 		t.Error("Got availability when some was expected in AvailabilityJSON")
// 	}

// }

// func TestRepository_BookRoom(t *testing.T) {

// 	req, _ := http.NewRequest("GET", "/book-room?id=1&start=2040-11-05&end=2040-11-22", nil)
// 	ctx := getCtx(req)
// 	req = req.WithContext(ctx)
// 	rr := httptest.NewRecorder()
// 	handler := http.HandlerFunc(Repo.BookRoom)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusSeeOther {
// 		t.Errorf("BookRoom handler failed: got %d, wanted %d", rr.Code, http.StatusSeeOther)
// 	}

// 	// test failure of start date
// 	req, _ = http.NewRequest("GET", "/book-room?id=1&start=invalid&end=2040-11-22", nil)
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	rr = httptest.NewRecorder()
// 	handler = http.HandlerFunc(Repo.BookRoom)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("BookRoom handler failed start date: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	// test failure of end date
// 	req, _ = http.NewRequest("GET", "/book-room?id=1&start=2040-11-05&end=invalid", nil)
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	rr = httptest.NewRecorder()
// 	handler = http.HandlerFunc(Repo.BookRoom)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("BookRoom handler failed end date: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	// test failure of room id
// 	req, _ = http.NewRequest("GET", "/book-room?id=22&start=2040-11-05&end=2040-11-22", nil)
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	rr = httptest.NewRecorder()
// 	handler = http.HandlerFunc(Repo.BookRoom)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("BookRoom handler failed room id: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}
// }

// func TestRepository_ChooseRoom(t *testing.T) {

// 	/*****************************************
// 	// first case -- reservation in session
// 	*****************************************/
// 	reservation := models.Reservation{
// 		RoomID: 1,
// 		Room: models.Room{
// 			ID:       1,
// 			RoomName: "General's Quarters",
// 		},
// 	}

// 	req, _ := http.NewRequest("GET", "/choose-room/1", nil)
// 	ctx := getCtx(req)
// 	req = req.WithContext(ctx)
// 	// set the RequestURI on the request so that we can grab the ID
// 	// from the URL
// 	req.RequestURI = "/choose-room/1"

// 	rr := httptest.NewRecorder()
// 	session.Put(ctx, "reservation", reservation)

// 	handler := http.HandlerFunc(Repo.ChooseRoom)

// 	handler.ServeHTTP(rr, req)

// 	if rr.Code != http.StatusSeeOther {
// 		t.Errorf("ChooseRoom handler in session returned wrong response code: got %d, wanted %d", rr.Code, http.StatusSeeOther)
// 	}

// 	// test not in session
// 	req, _ = http.NewRequest("GET", "/choose-room/1", nil)
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	// set the RequestURI on the request so that we can grab the ID
// 	// from the URL
// 	req.RequestURI = "/choose-room/1"
// 	rr = httptest.NewRecorder()
// 	handler = http.HandlerFunc(Repo.ChooseRoom)

// 	handler.ServeHTTP(rr, req)

// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("ChooseRoom handler not in session returned wrong response code: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	// test invalid id
// 	req, _ = http.NewRequest("GET", "/choose-room/invalid", nil)
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	req.RequestURI = "/choose-room/invalid"
// 	rr = httptest.NewRecorder()
// 	handler = http.HandlerFunc(Repo.ChooseRoom)

// 	handler.ServeHTTP(rr, req)

// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("ChooseRoom handler invalid id returned wrong response code: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}
// }

// func TestRepository_PostSearchAvailability(t *testing.T) {

// 	// test invalid parse form
// 	req, _ := http.NewRequest("POST", "/search-availability", nil)
// 	ctx := getCtx(req)
// 	req = req.WithContext(ctx)

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	rr := httptest.NewRecorder()

// 	handler := http.HandlerFunc(Repo.PostSearchAvailability)
// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostSearchAvailability handler returned wrong response code: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	// test invalid start date
// 	// reqBody := "start=invalid"
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "end=2023-11-23")

// 	postedData := url.Values{}
// 	postedData.Add("start", "invalid")

// 	req, _ = http.NewRequest("POST", "/search-availability", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	rr = httptest.NewRecorder()

// 	handler = http.HandlerFunc(Repo.PostSearchAvailability)
// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostSearchAvailability handler start date returned wrong response code: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	// test invalid end date
// 	// reqBody = "start=2023-11-22"
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "end=invalid")

// 	postedData = url.Values{}
// 	postedData.Add("start", "2023-11-22")

// 	req, _ = http.NewRequest("POST", "/search-availability", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	rr = httptest.NewRecorder()

// 	handler = http.HandlerFunc(Repo.PostSearchAvailability)
// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostSearchAvailability handler end date returned wrong response code: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	// test room not available
// 	// reqBody = "start=2060-01-01"
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "end=2023-11-23")

// 	postedData = url.Values{}
// 	postedData.Add("start", "2060-01-01")

// 	req, _ = http.NewRequest("POST", "/search-availability", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	rr = httptest.NewRecorder()

// 	handler = http.HandlerFunc(Repo.PostSearchAvailability)
// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostSearchAvailability handler room not available returned wrong response code: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	// test room 0
// 	// reqBody = "start=2050-12-31"
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "end=2023-11-23")

// 	postedData = url.Values{}
// 	postedData.Add("start", "2050-12-31")
// 	postedData.Add("end", "2023-11-23")

// 	req, _ = http.NewRequest("POST", "/search-availability", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	rr = httptest.NewRecorder()

// 	handler = http.HandlerFunc(Repo.PostSearchAvailability)
// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusSeeOther {
// 		t.Errorf("PostSearchAvailability handler room not available returned wrong response code: got %d, wanted %d", rr.Code, http.StatusSeeOther)
// 	}

// 	// test room available
// 	// reqBody = "start=2020-12-31"
// 	// reqBody = fmt.Sprintf("%s&%s", reqBody, "end=2023-11-23")

// 	postedData = url.Values{}
// 	var key string = "start"
// 	postedData[key] = append(postedData[key], "2020-12-31")
// 	postedData.Add("end", "2023-11-23")

// 	req, _ = http.NewRequest("POST", "/search-availability", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	rr = httptest.NewRecorder()

// 	handler = http.HandlerFunc(Repo.PostSearchAvailability)
// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusOK {
// 		t.Errorf("PostSearchAvailability handler room not available returned wrong response code: got %d, wanted %d", rr.Code, http.StatusOK)
// 	}

// }

// func TestRepository_PostContact(t *testing.T) {
// 	req, _ := http.NewRequest("POST", "/contact", nil)
// 	ctx := getCtx(req)
// 	req = req.WithContext(ctx)

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	rr := httptest.NewRecorder()

// 	handler := http.HandlerFunc(Repo.PostContact)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusOK {
// 		t.Errorf("PostContact handler returned wrong response code: got %d, wanted %d", rr.Code, http.StatusOK)
// 	}
// }

// func TestRepository_GetShowLogin(t *testing.T) {
// 	req, _ := http.NewRequest("GET", "/user/login", nil)
// 	ctx := getCtx(req)
// 	req = req.WithContext(ctx)
// 	rr := httptest.NewRecorder()
// 	handler := http.HandlerFunc(Repo.GetShowLogin)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusOK {
// 		t.Errorf("ShowLogin handler failed: got %d, wanted %d", rr.Code, http.StatusOK)
// 	}

// }

// // table driven test
// var loginTests = []struct {
// 	name               string
// 	email              string
// 	expectedStatusCode int
// 	expectedHTML       string
// 	expectedLocation   string
// }{
// 	{
// 		"valid-credentials",
// 		"valid@here.ca",
// 		http.StatusSeeOther,
// 		"",
// 		"/",
// 	},
// 	{
// 		"invalid-credentials",
// 		"invalid@gmail.com",
// 		http.StatusSeeOther,
// 		"",
// 		"/user/login",
// 	},
// 	{
// 		"invalid-data",
// 		"j",
// 		http.StatusOK,
// 		`action="/user/login"`,
// 		"",
// 	},
// }

// func TestLogin(t *testing.T) {
// 	for _, e := range loginTests {
// 		postedData := url.Values{}
// 		postedData.Add("email", e.email)
// 		postedData.Add("password", "password")

// 		// create request
// 		req, _ := http.NewRequest("POST", "/user/login", strings.NewReader(postedData.Encode()))
// 		ctx := getCtx(req)
// 		req = req.WithContext(ctx)

// 		// set the header
// 		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
// 		rr := httptest.NewRecorder()

// 		// call the handler
// 		handler := http.HandlerFunc(Repo.PostShowLogin)
// 		handler.ServeHTTP(rr, req)
// 		if rr.Code != e.expectedStatusCode {
// 			t.Errorf("TestLogin handler returned wrong response code: got %d, expexted code %d", rr.Code, e.expectedStatusCode)
// 		}

// 		if e.expectedLocation != "" {
// 			// get the URL from test
// 			actualLocation, _ := rr.Result().Location()
// 			if actualLocation.String() != e.expectedLocation {
// 				t.Errorf("failed %s: expected location %s, but got location %s", e.name, e.expectedLocation, actualLocation.String())
// 			}
// 		}

// 		// check for expexted values in HTML
// 		if e.expectedHTML != "" {
// 			// read the response body into a string
// 			html := rr.Body.String()

// 			if !strings.Contains(html, e.expectedHTML) {
// 				t.Errorf("failed %s: expected html to find %s, but did not", e.name, e.expectedHTML)
// 			}
// 		}
// 	}
// }

// func __TestRepository_PostShowLogin(t *testing.T) {

// 	//invalid form data
// 	postedData := url.Values{}
// 	postedData.Add("email", "test@gmail")
// 	postedData.Add("password", "123456")

// 	req, _ := http.NewRequest("POST", "/user/login", nil)
// 	ctx := getCtx(req)
// 	req = req.WithContext(ctx)
// 	rr := httptest.NewRecorder()

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	handler := http.HandlerFunc(Repo.PostShowLogin)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostShowLogin handler failed: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	//valid form data
// 	postedData = url.Values{}
// 	postedData.Add("email", "test@gmail.com")
// 	postedData.Add("password", "123456")

// 	req, _ = http.NewRequest("POST", "/user/login", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	rr = httptest.NewRecorder()

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	handler = http.HandlerFunc(Repo.PostShowLogin)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusSeeOther {
// 		t.Errorf("PostShowLogin handler failed: got %d, wanted %d", rr.Code, http.StatusSeeOther)
// 	}

// 	// invalid email
// 	postedData = url.Values{}
// 	postedData.Add("email", "invalid")

// 	req, _ = http.NewRequest("POST", "/user/login", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	rr = httptest.NewRecorder()

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	handler = http.HandlerFunc(Repo.PostShowLogin)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusOK {
// 		t.Errorf("PostShowLogin invalid email handler failed: got %d, wanted %d", rr.Code, http.StatusOK)
// 	}

// }

// func __TestRepository_AdminPostShowReservation(t *testing.T) {

// 	// invalid form
// 	req, _ := http.NewRequest("POST", "/admin/reservations/cal/19", nil)
// 	ctx := getCtx(req)
// 	req = req.WithContext(ctx)

// 	rr := httptest.NewRecorder()

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	handler := http.HandlerFunc(Repo.AdminPostShowReservation)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostShowLogin handler failed: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	//valid form data
// 	postedData := url.Values{}
// 	postedData.Add("first_name", "Jericho")
// 	postedData.Add("last_name", "Baisa")
// 	postedData.Add("email", "test@gmail.com")
// 	postedData.Add("phone", "123456789")

// 	req, _ = http.NewRequest("POST", "/admin/admin/reservations/cal/9", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	req.RequestURI = "/admin/reservations/cal/19"
// 	rr = httptest.NewRecorder()

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	handler = http.HandlerFunc(Repo.AdminPostShowReservation)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusSeeOther {
// 		t.Errorf("PostShowLogin handler failed: got %d, wanted %d", rr.Code, http.StatusSeeOther)
// 	}

// 	//invalid id
// 	postedData = url.Values{}
// 	postedData.Add("first_name", "Jericho")
// 	postedData.Add("last_name", "Baisa")
// 	postedData.Add("email", "test@gmail.com")
// 	postedData.Add("phone", "123456789")

// 	req, _ = http.NewRequest("POST", "/admin/admin/reservations/cal/200", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	req.RequestURI = "/admin/reservations/cal/invalid"
// 	rr = httptest.NewRecorder()

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	handler = http.HandlerFunc(Repo.AdminPostShowReservation)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostShowLogin handler failed: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// 	//invalid id
// 	postedData = url.Values{}
// 	postedData.Add("first_name", "Jericho")
// 	postedData.Add("last_name", "Baisa")
// 	postedData.Add("email", "test@gmail.com")
// 	postedData.Add("phone", "123456789")

// 	req, _ = http.NewRequest("POST", "/admin/admin/reservations/cal/200", strings.NewReader(postedData.Encode()))
// 	ctx = getCtx(req)
// 	req = req.WithContext(ctx)
// 	req.RequestURI = "/admin/reservations/cal/200"
// 	rr = httptest.NewRecorder()

// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

// 	handler = http.HandlerFunc(Repo.AdminPostShowReservation)

// 	handler.ServeHTTP(rr, req)
// 	if rr.Code != http.StatusTemporaryRedirect {
// 		t.Errorf("PostShowLogin handler failed: got %d, wanted %d", rr.Code, http.StatusTemporaryRedirect)
// 	}

// }

// var reservationsTests = []struct {
// 	name               string
// 	email              string
// 	expectedStatusCode int
// 	expectedHTML       string
// 	expectedLocation   string
// }{
// 	{
// 		"invalid-post-reservation",
// 		"valid@here.ca",
// 		http.StatusTemporaryRedirect,
// 		"",
// 		"/",
// 	},
// }

// func TestReservations(t *testing.T) {
// 	for _, e := range reservationsTests {
// 		postedData := url.Values{}
// 		postedData.Add("email", e.email)
// 		postedData.Add("password", "password")

// 		// create request
// 		req, _ := http.NewRequest("POST", "/admin/reservations/calendar/1", strings.NewReader(postedData.Encode()))
// 		ctx := getCtx(req)
// 		req = req.WithContext(ctx)

// 		// set the header
// 		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
// 		rr := httptest.NewRecorder()

// 		// call the handler
// 		handler := http.HandlerFunc(Repo.PostShowLogin)
// 		handler.ServeHTTP(rr, req)
// 		if rr.Code != e.expectedStatusCode {
// 			t.Errorf("TestLogin handler returned wrong response code: got %d, expexted code %d", rr.Code, e.expectedStatusCode)
// 		}

// 		if e.expectedLocation != "" {
// 			// get the URL from test
// 			actualLocation, _ := rr.Result().Location()
// 			if actualLocation.String() != e.expectedLocation {
// 				t.Errorf("failed %s: expected location %s, but got location %s", e.name, e.expectedLocation, actualLocation.String())
// 			}
// 		}

// 		// check for expexted values in HTML
// 		if e.expectedHTML != "" {
// 			// read the response body into a string
// 			html := rr.Body.String()

// 			if !strings.Contains(html, e.expectedHTML) {
// 				t.Errorf("failed %s: expected html to find %s, but did not", e.name, e.expectedHTML)
// 			}
// 		}
// 	}
// }

// func getCtx(req *http.Request) context.Context {
// 	ctx, err := session.Load(req.Context(), req.Header.Get("X-Session"))
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	return ctx
// }
