package main

import (
	"net/http"

	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/config"
	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func routes(app *config.AppConfig) http.Handler {
	// mux := pat.New()

	// mux.Get("/", http.HandlerFunc(handlers.Repo.Home))
	// mux.Get("/about", http.HandlerFunc(handlers.Repo.About))

	mux := chi.NewRouter()

	mux.Use(middleware.Recoverer)
	mux.Use(NoSurf)
	mux.Use(SessionLoad)

	mux.Get("/", handlers.Repo.Home)
	mux.Get("/about", handlers.Repo.About)
	mux.Get("/rooms/general-quarters", handlers.Repo.RoomGeneralsQuarter)
	mux.Get("/rooms/major-quarters", handlers.Repo.RoomMajorQuarter)

	mux.Get("/search-availability", handlers.Repo.GetSearchAvailability)
	mux.Post("/search-availability", handlers.Repo.PostSearchAvailability)
	mux.Post("/search-availability-json", handlers.Repo.SearchAvailabilityJSON)
	mux.Get("/choose-room/{id}", handlers.Repo.ChooseRoom)
	mux.Get("/book-room", handlers.Repo.BookRoom)

	mux.Get("/make-reservation", handlers.Repo.GetMakeReservation)
	mux.Post("/make-reservation", handlers.Repo.PostMakeReservation)
	mux.Get("/reservation-summary", handlers.Repo.ReservationSummary)

	mux.Get("/user/login", handlers.Repo.GetShowLogin)
	mux.Post("/user/login", handlers.Repo.PostShowLogin)
	mux.Get("/user/logout", handlers.Repo.Logout)

	mux.Get("/contact", handlers.Repo.GetContact)
	mux.Post("/contact", handlers.Repo.PostContact)

	fileServer := http.FileServer(http.Dir("./static/"))
	mux.Handle("/static/*", http.StripPrefix("/static", fileServer))

	mux.Route("/admin", func(mux chi.Router) {
		// mux.Use(Auth)
		mux.Get("/dashboard", handlers.Repo.AdminDashboard)

		mux.Get("/reservations-new", handlers.Repo.AdminNewReservations)
		mux.Get("/reservations-all", handlers.Repo.AdminAllReservations)

		mux.Get("/reservations-calendar", handlers.Repo.AdminReservationsCalendar)
		mux.Post("/reservations-calendar", handlers.Repo.AdminPostReservationsCalendar)

		mux.Get("/process-reservation/{src}/{id}/do", handlers.Repo.AdminProcessReservation)
		mux.Get("/delete-reservation/{src}/{id}/do", handlers.Repo.AdminDeleteReservation)

		mux.Get("/reservations/{src}/{id}/show", handlers.Repo.AdminShowReservation)
		mux.Post("/reservations/{src}/{id}", handlers.Repo.AdminPostShowReservation)

	})

	return mux
}
