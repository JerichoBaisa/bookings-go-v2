package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/config"
	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/driver"
	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/forms"
	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/helpers"
	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/models"
	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/render"
	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/repository"
	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/repository/dbrepo"
	"github.com/go-chi/chi/v5"
)

// Repo the repository used by the handlers
var Repo *Repository

// Repository is the repository type
type Repository struct {
	App *config.AppConfig
	DB  repository.DatabaseRepo
}

// NewRepo create a new repository
func NewRepo(a *config.AppConfig, db *driver.DB) *Repository {
	return &Repository{
		App: a,
		DB:  dbrepo.NewPostgresRepo(db.SQL, a),
	}
}

// NewRepo create a new repository
func NewTestRepo(a *config.AppConfig) *Repository {
	return &Repository{
		App: a,
		DB:  dbrepo.NewTestingRepo(a),
	}
}

// NewHandlers set the repository for the handler
func NewHandlers(r *Repository) {
	Repo = r
}

func (m *Repository) Home(w http.ResponseWriter, r *http.Request) {
	// remoteIP := r.RemoteAddr
	// m.App.Session.Put(r.Context(), "remote_ip", remoteIP)

	render.Template(w, r, "home.page.html", &models.TemplateData{})
}

// About is the about page handler
func (m *Repository) About(w http.ResponseWriter, r *http.Request) {
	// perform some logic

	render.Template(w, r, "about.page.html", &models.TemplateData{})

}

func (m *Repository) RoomGeneralsQuarter(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "general-quarters.page.html", &models.TemplateData{})
}

func (m *Repository) RoomMajorQuarter(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "major-quarters.page.html", &models.TemplateData{})
}

func (m *Repository) GetMakeReservation(w http.ResponseWriter, r *http.Request) {

	res, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)
	if !ok {
		// helpers.ServerError(w, errors.New("cannot get reservation from session"))
		m.App.Session.Put(r.Context(), "error", "can't get reservation from session get make reservation")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	room, err := m.DB.GetRoomByID(res.RoomID)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't find room!")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	res.Room = room

	m.App.Session.Put(r.Context(), "reservation", res)

	sd := res.StartDate.Format("2006-01-02")
	ed := res.EndDate.Format("2006-01-02")

	stringMap := make(map[string]string)
	stringMap["start_date"] = sd
	stringMap["end_date"] = ed

	data := make(map[string]interface{})
	data["reservation"] = res
	render.Template(w, r, "make-reservation.page.html", &models.TemplateData{
		Form:      forms.New(nil),
		Data:      data,
		StringMap: stringMap,
	})
}

// PostMakeReservation handles a post reservation form
func (m *Repository) PostMakeReservation(w http.ResponseWriter, r *http.Request) {

	reservation, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)

	if !ok {
		m.App.Session.Put(r.Context(), "error", "can't get reservation from session post make reservation")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse form")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	reservation.FirstName = r.Form.Get("first_name")
	reservation.LastName = r.Form.Get("last_name")
	reservation.Email = r.Form.Get("email")
	reservation.Phone = r.Form.Get("phone")

	form := forms.New(r.PostForm)

	// form.Has("first_name", r)
	form.Required("first_name", "last_name", "email")
	form.MinLength("first_name", 3)
	form.IsEmail("email")

	if !form.Valid() {
		data := make(map[string]interface{})
		data["reservation"] = reservation
		m.App.Session.Put(r.Context(), "error", "invalid form data")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)

		render.Template(w, r, "make-reservation.page.html", &models.TemplateData{
			Form: form,
			Data: data,
		})

		return

	}

	newReservationID, err := m.DB.InsertReservation(reservation)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't insert reservation into database")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	roomRestriction := models.RoomRestriction{
		StartDate:     reservation.StartDate,
		Enddate:       reservation.EndDate,
		RoomID:        reservation.RoomID,
		ReservationID: newReservationID,
		RestrictionID: 1,
	}

	err = m.DB.InsertRoomRestriction(roomRestriction)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't insert room restriction into database")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	// send notification - first to guest
	htmlMessage := fmt.Sprintf(`
	<strong>Reservation Confirmation</strong><br>
	<p>Dear %s, </p>
	<p>This is confirm your reservation from %s to %s.</p>
	`, reservation.FirstName, reservation.StartDate.Format("2006-01-02"), reservation.EndDate.Format("2006-01-02"))
	msg := models.MailData{
		To:       reservation.Email,
		From:     "developer@subhub.com",
		Subject:  "Reservation Confirmation",
		Content:  htmlMessage,
		Template: "basic.html",
	}

	m.App.MailChan <- msg

	// send notification to property owner
	htmlMessage = fmt.Sprintf(`
		<strong>Reservation Notification</strong><br>
		<p>A reservation has been made for %s from %s to %s.</p>
		`, reservation.FirstName, reservation.StartDate.Format("2006-01-02"), reservation.EndDate.Format("2006-01-02"))

	msg = models.MailData{
		To:      "developer@subhub.com",
		From:    "developer@subhub.com",
		Subject: "Reservation Confirmation",
		Content: htmlMessage,
	}

	m.App.MailChan <- msg

	// messagingin standard library
	// from := "jericho@go.com"
	// auth := smtp.PlainAuth("", from, "", "localhost")
	// err = smtp.SendMail("localhost:1025", auth, from, []string{"abaisa.it@gmail.com"}, []byte("Hello, World!"))
	// if err != nil {
	// 	log.Println(err)
	// }

	m.App.Session.Put(r.Context(), "reservation", reservation)

	http.Redirect(w, r, "/reservation-summary", http.StatusSeeOther)

}

// ReservationSummary display the reservation summary page
func (m *Repository) ReservationSummary(w http.ResponseWriter, r *http.Request) {
	reservation, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)
	if !ok {
		m.App.Session.Put(r.Context(), "error", "Can't get resevation from session")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	m.App.Session.Remove(r.Context(), "reservation")
	data := make(map[string]interface{})
	data["reservation"] = reservation

	sd := reservation.StartDate.Format("2006-01-02")
	ed := reservation.EndDate.Format("2006-01-02")

	stringMap := make(map[string]string)
	stringMap["start_date"] = sd
	stringMap["end_date"] = ed

	render.Template(w, r, "reservation-summary.page.html", &models.TemplateData{
		Data:      data,
		StringMap: stringMap,
	})
}

// GetSearchAvailability renders serach availability page
func (m *Repository) GetSearchAvailability(w http.ResponseWriter, r *http.Request) {

	render.Template(w, r, "search-availability.page.html", &models.TemplateData{})
}

// PostSearchAvailability renders search availability page
func (m *Repository) PostSearchAvailability(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse form")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	start := r.Form.Get("start")
	end := r.Form.Get("end")

	// 2020-01-01 --- 01/02 03:04:05PM '06 -0700

	// described my date layout
	layout := "2006-01-02"
	startDate, err := time.Parse(layout, start)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse start date ")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	endDate, err := time.Parse(layout, end)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse end date ")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	rooms, err := m.DB.SearchAvailabilityForAllRooms(startDate, endDate)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't connect to database SearchAvailabilityForAllRooms ")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	for _, i := range rooms {
		m.App.InfoLog.Println(i.ID, i.RoomName)
	}

	if len(rooms) == 0 {
		m.App.InfoLog.Println("No Availability")
		m.App.Session.Put(r.Context(), "error", "No availability")
		http.Redirect(w, r, "/search-availability", http.StatusSeeOther)
		return
	}

	data := make(map[string]interface{})
	data["rooms"] = rooms

	res := models.Reservation{
		StartDate: startDate,
		EndDate:   endDate,
	}

	m.App.Session.Put(r.Context(), "reservation", res)
	render.Template(w, r, "choose-room.page.html", &models.TemplateData{
		Data: data,
	})

}

// ChooseRoom displays list of all available rooms
func (m *Repository) ChooseRoom(w http.ResponseWriter, r *http.Request) {
	// roomId, err := strconv.Atoi(chi.URLParam(r, "id"))
	exploded := strings.Split(r.RequestURI, "/")
	roomId, err := strconv.Atoi(exploded[2])
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't string convert room id ")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	res, ok := m.App.Session.Get(r.Context(), "reservation").(models.Reservation)
	if !ok {
		m.App.Session.Put(r.Context(), "error", "can't get session data ")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	res.RoomID = roomId

	m.App.Session.Put(r.Context(), "reservation", res)

	http.Redirect(w, r, "/make-reservation", http.StatusSeeOther)

}

type jsonResponse struct {
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
	RoomID    string `json:"room_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// SearchAvailabilityJSON handles request for availability and send JSON response
func (m *Repository) SearchAvailabilityJSON(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		//can't parse form so return appropriate json
		resp := jsonResponse{
			OK:      false,
			Message: "Internal server error",
		}

		out, _ := json.MarshalIndent(resp, "", "    ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
		return
		// m.App.Session.Put(r.Context(), "error", "can't parse form")
		// http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}

	sd := r.Form.Get("start")
	ed := r.Form.Get("end")

	layout := "2006-01-02"
	startDate, err := time.Parse(layout, sd)
	if err != nil {
		resp := jsonResponse{
			OK:      false,
			Message: "Error parse start date SearchAvailabilityJSON",
		}

		out, _ := json.MarshalIndent(resp, "", "    ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
		return
	}
	endDate, err := time.Parse(layout, ed)
	if err != nil {
		resp := jsonResponse{
			OK:      false,
			Message: "Error parse end date SearchAvailabilityJSON",
		}

		out, _ := json.MarshalIndent(resp, "", "    ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
		return
	}

	roomID, err := strconv.Atoi(r.Form.Get("room_id"))
	if err != nil {
		resp := jsonResponse{
			OK:      false,
			Message: "Error string convertion room id SearchAvailabilityJSON",
		}

		out, _ := json.MarshalIndent(resp, "", "    ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
		return
	}

	isOk, err := m.DB.SearchAvailabilityByDatesByRoomID(startDate, endDate, roomID)
	if err != nil {
		resp := jsonResponse{
			OK:      false,
			Message: "Error connecting to database",
		}

		out, _ := json.MarshalIndent(resp, "", "    ")
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
		return
	}

	message := "Room is Available"
	if !isOk {
		message = "Room is not available on given date"
	}

	resp := jsonResponse{
		OK:        isOk,
		Message:   message,
		RoomID:    strconv.Itoa(roomID),
		StartDate: sd,
		EndDate:   ed,
	}

	out, _ := json.MarshalIndent(resp, "", "     ")

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)

}

// BookRoom take url paramenter build a session variable and take user to make reserservation
func (m *Repository) BookRoom(w http.ResponseWriter, r *http.Request) {
	roomID, _ := strconv.Atoi(r.URL.Query().Get("id"))

	sd := r.URL.Query().Get("start")
	ed := r.URL.Query().Get("end")

	layout := "2006-01-02"
	startDate, err := time.Parse(layout, sd)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse start date ")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	endDate, err := time.Parse(layout, ed)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse end date ")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	var res models.Reservation

	room, err := m.DB.GetRoomByID(roomID)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't connect to a database BookRoom")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	res.Room.RoomName = room.RoomName
	res.RoomID = roomID
	res.StartDate = startDate
	res.EndDate = endDate

	m.App.Session.Put(r.Context(), "reservation", res)

	http.Redirect(w, r, "/make-reservation", http.StatusSeeOther)

}

// Login
func (m *Repository) GetShowLogin(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "login.page.html", &models.TemplateData{
		Form: forms.New(nil),
	})
}

// PostShowLogin handles user logging the user in
func (m *Repository) PostShowLogin(w http.ResponseWriter, r *http.Request) {
	_ = m.App.Session.RenewToken(r.Context())

	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse login form")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	email := r.Form.Get("email")
	testPassword := r.Form.Get("password")

	form := forms.New(r.PostForm)
	form.Required("email", "password")
	form.IsEmail("email")
	if !form.Valid() {
		// TODO - take user back to login page
		render.Template(w, r, "login.page.html", &models.TemplateData{
			Form: form,
		})
		return
	}

	uId, msg, err := m.DB.Authenticate(email, testPassword)
	if err != nil {
		m.App.Session.Put(r.Context(), "error", msg)
		http.Redirect(w, r, "/user/login", http.StatusSeeOther)
		return
	}

	m.App.Session.Put(r.Context(), "user_id", uId)
	m.App.Session.Put(r.Context(), "flash", "Logged in successfully")

	http.Redirect(w, r, "/", http.StatusSeeOther)

	//
	// _, err = m.DB.GetUserByID(uId)
	// if err != nil {
	// 	m.App.Session.Put(r.Context(), "error", "erro in getuser by id")
	// 	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	// 	return
	// }

	// render.Template(w, r, "login.page.html", &models.TemplateData{
	// 	Form: forms.New(nil),
	// })
}

// Logout logs a user out
func (m *Repository) Logout(w http.ResponseWriter, r *http.Request) {
	_ = m.App.Session.Destroy(r.Context())
	_ = m.App.Session.RenewToken(r.Context())

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

// AdminDashboard
func (m *Repository) AdminDashboard(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "admin-dashboard.page.html", &models.TemplateData{})
}

// AdminNewReservations shows all new reservations in admin tool
func (m *Repository) AdminNewReservations(w http.ResponseWriter, r *http.Request) {

	res, err := m.DB.GetAdminAllNewReservations()
	if err != nil {
		log.Println(err)
		m.App.Session.Put(r.Context(), "error", "can't connect to a database new reservations")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	data := make(map[string]interface{})
	data["new_reservation"] = res
	render.Template(w, r, "admin-new-reservations.page.html", &models.TemplateData{
		Data: data,
	})
}

// AdminAllNewReservations shows all reservations in admin tool
func (m *Repository) AdminAllReservations(w http.ResponseWriter, r *http.Request) {

	res, err := m.DB.GetAdminAllReservations()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't connect to a database all reservations")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	data := make(map[string]interface{})
	data["reservation"] = res
	render.Template(w, r, "admin-all-reservations.page.html", &models.TemplateData{
		Data: data,
	})
}

func (m *Repository) AdminShowReservation(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	id, err := strconv.Atoi(exploded[4])
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "problem with show reservations")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}

	src := exploded[3]

	stringMap := make(map[string]string)
	stringMap["src"] = src

	year := r.URL.Query().Get("y")
	month := r.URL.Query().Get("m")

	stringMap["month"] = month
	stringMap["year"] = year

	res, err := m.DB.GetReservationByID(id)
	if err != nil {
		log.Println(err)
		m.App.Session.Put(r.Context(), "error", "can't connect database GetReservationByID")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}

	data := make(map[string]interface{})
	data["reservation"] = res
	render.Template(w, r, "admin-reservations-show.page.html", &models.TemplateData{
		Data:      data,
		StringMap: stringMap,
		Form:      forms.New(nil),
	})

}

func (m *Repository) AdminPostShowReservation(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "can't parse form")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
	exploded := strings.Split(r.RequestURI, "/")
	id, err := strconv.Atoi(exploded[4])
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "problem with show reservations")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}

	src := exploded[3]

	stringMap := make(map[string]string)
	stringMap["src"] = src

	res, err := m.DB.GetReservationByID(id)
	if err != nil {
		log.Println(err)
		m.App.Session.Put(r.Context(), "error", "can't connect database AdminPostShowReservation")
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}

	res.FirstName = r.Form.Get("first_name")
	res.LastName = r.Form.Get("last_name")
	res.Email = r.Form.Get("email")
	res.Phone = r.Form.Get("phone")

	err = m.DB.UpdateReservation(res, id)
	if err != nil {
		log.Println(err)
		helpers.ServerError(w, err)
		return
	}

	month := r.Form.Get("month")
	year := r.Form.Get("year")

	m.App.Session.Put(r.Context(), "flash", "Changes saved")

	if year == "" {
		http.Redirect(w, r, fmt.Sprintf("/admin/reservations-%s", src), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/admin/reservations-%s?y=%s&m=%s", src, year, month), http.StatusSeeOther)
	}

	// m.DB.UpdateProcessedForReservation(id, 1)

}

// AdminProcessReservation marks a reservation as a processed
func (m *Repository) AdminProcessReservation(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	src := chi.URLParam(r, "src")

	err := m.DB.UpdateProcessedForReservation(id, 1)
	if err != nil {
		log.Println(err)
	}

	year := r.URL.Query().Get("y")
	month := r.URL.Query().Get("m")

	m.App.Session.Put(r.Context(), "flash", "Reservation marked as processed")

	if year == "" {
		http.Redirect(w, r, fmt.Sprintf("/admin/reservations-%s", src), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/admin/reservations-calendar?y=%s&m=%s", year, month), http.StatusSeeOther)
	}

}

// AdminDeleteReservation deletes a reservation
func (m *Repository) AdminDeleteReservation(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))
	src := chi.URLParam(r, "src")

	err := m.DB.DeleteReservation(id)
	if err != nil {
		log.Println(err)
	}

	year := r.URL.Query().Get("y")
	month := r.URL.Query().Get("m")

	m.App.Session.Put(r.Context(), "flash", "Reservation deleted")

	if year == "" {
		http.Redirect(w, r, fmt.Sprintf("/admin/reservations-%s", src), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/admin/reservations-calendar?y=%s&m=%s", year, month), http.StatusSeeOther)
	}

}

// AdminReservationCalendar displays the reservation calendar
func (m *Repository) AdminReservationsCalendar(w http.ResponseWriter, r *http.Request) {
	// assume there is no month/year specified
	now := time.Now()

	if r.URL.Query().Get("y") != "" {
		year, _ := strconv.Atoi(r.URL.Query().Get("y"))
		month, _ := strconv.Atoi(r.URL.Query().Get("m"))

		now = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	}

	data := make(map[string]interface{})
	data["now"] = now

	next := now.AddDate(0, 1, 0)
	last := now.AddDate(0, -1, 0)

	nextMonth := next.Format("01")       // 01 give us two digit month
	nextMonthYear := next.Format("2006") // 2006 give us four digit year

	lastMonth := last.Format("01")       // 01 give us two digit month
	lastMonthYear := last.Format("2006") // 2006 give us four digit year

	stringMap := make(map[string]string)
	stringMap["next_month"] = nextMonth
	stringMap["next_month_year"] = nextMonthYear

	stringMap["last_month"] = lastMonth
	stringMap["last_month_year"] = lastMonthYear

	stringMap["this_month"] = now.Format("01")
	stringMap["this_month_year"] = now.Format("2006")

	//get the first and last day of the month
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()
	firstOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1) // -1 last day of the current month

	intMap := make(map[string]int)
	intMap["days_in_month"] = lastOfMonth.Day()

	rooms, err := m.DB.AllRooms()
	if err != nil {
		log.Println(err)
		helpers.ServerError(w, err)
		return
	}

	data["rooms"] = rooms

	for _, x := range rooms {
		// create maps - this will holds data to avoid multiple checking in the database
		reservationMap := make(map[string]int)
		blockMap := make(map[string]int)

		for d := firstOfMonth; !d.After(lastOfMonth); d = d.AddDate(0, 0, 1) {
			reservationMap[d.Format("2006-01-2")] = 0
			blockMap[d.Format("2006-01-2")] = 0
		}

		// get all restriction for the current room
		restrictions, err := m.DB.GetRestrictionsForRoomByDate(x.ID, firstOfMonth, lastOfMonth)
		if err != nil {
			helpers.ServerError(w, err)
			return
		}

		for _, y := range restrictions {
			if y.ReservationID > 0 {
				// it's a reservation
				for d := y.StartDate; !d.After(y.Enddate); d = d.AddDate(0, 0, 1) {
					reservationMap[d.Format("2006-01-2")] = y.ReservationID
				}
			} else {
				// it's a block
				blockMap[y.StartDate.Format("2006-01-2")] = y.ID
			}
		}

		data[fmt.Sprintf("reservation_map_%d", x.ID)] = reservationMap
		data[fmt.Sprintf("block_map_%d", x.ID)] = blockMap

		m.App.Session.Put(r.Context(), fmt.Sprintf("block_map_%d", x.ID), blockMap)
	}

	render.Template(w, r, "admin-reservations-calendar.page.html", &models.TemplateData{
		StringMap: stringMap,
		Data:      data,
		IntMap:    intMap,
	})
}

// AdminPostReservationsCalendar handles post of reservation calendar
func (m *Repository) AdminPostReservationsCalendar(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	year, _ := strconv.Atoi(r.Form.Get("y"))
	month, _ := strconv.Atoi(r.Form.Get("m"))

	//process blocks
	rooms, err := m.DB.AllRooms()
	if err != nil {
		helpers.ServerError(w, err)
		return
	}

	form := forms.New(r.PostForm)

	for _, x := range rooms {
		// Get the block map from the session. Loop through the entire map, if we have an entry in the map
		// that does not exists in pur posted data, and if the restriciton id > 0, then its a block we need to remove.
		curMap := m.App.Session.Get(r.Context(), fmt.Sprintf("block_map_%d", x.ID)).(map[string]int)
		for name, value := range curMap {
			// ok will be false if the value is not in the map
			if val, ok := curMap[name]; ok {
				// only pay attention to values >0 and that are not in the form post
				// the rest are just placeholder for days without blocks
				if val > 0 {
					if !form.Has(fmt.Sprintf("remove_block_%d_%s", x.ID, name)) {
						log.Println("would delete block=value:", value, " x.ID: ", x.ID, " Name:", name)
						err := m.DB.DeleteBlockByID(value)
						if err != nil {
							log.Println(err)
						}

					}
				}

			}
		}
	}

	// now handle new blocks

	for name, _ := range r.PostForm {
		// log.Println("Form has name:", name)
		if strings.HasPrefix(name, "add_block") {
			exploded := strings.Split(name, "_")
			roomID, _ := strconv.Atoi(exploded[2])
			t, _ := time.Parse("2006-01-2", exploded[3])
			// insert a new block
			// log.Println("would insert block for room id", roomID, " for date ", exploded[3])
			err := m.DB.InsertBlockForRoom(roomID, t)
			if err != nil {
				log.Println(err)
			}
		}
	}

	m.App.Session.Put(r.Context(), "flash", "Changes saved")
	http.Redirect(w, r, fmt.Sprintf("/admin/reservations-calendar?y=%d&m=%d", year, month), http.StatusSeeOther)

}

// CONTACT PAGE
func (m *Repository) PostContact(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]interface{})
	contactData := models.Contact{FirstName: "Jericho",
		LastName: "Baisa",
		Email:    "jb@gmail.com",
		Phone:    "111-222-1122",
		Message:  "Hello World! from Contact",
	}

	data["contact"] = contactData
	render.Template(w, r, "contact.page.html", &models.TemplateData{
		Data: data,
	})
}

func (m *Repository) GetContact(w http.ResponseWriter, r *http.Request) {
	render.Template(w, r, "contact.page.html", &models.TemplateData{})
}
