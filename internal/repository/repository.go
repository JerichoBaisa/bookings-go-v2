package repository

import (
	"time"

	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/models"
)

type DatabaseRepo interface {
	AllUsers() bool
	InsertReservation(res models.Reservation) (int, error)
	InsertRoomRestriction(r models.RoomRestriction) error
	SearchAvailabilityByDatesByRoomID(start, end time.Time, roomID int) (bool, error)
	SearchAvailabilityForAllRooms(start, end time.Time) ([]models.Room, error)
	GetRoomByID(id int) (models.Room, error)

	GetUserByID(id int) (models.User, error)
	UpdateUser(u models.User) error
	Authenticate(email, testPassword string) (int, string, error)

	GetAdminAllReservations() ([]models.Reservation, error)
	GetAdminAllNewReservations() ([]models.Reservation, error)
	GetReservationByID(id int) (models.Reservation, error)

	UpdateReservation(u models.Reservation, id int) error
	DeleteReservation(id int) error
	UpdateProcessedForReservation(id, processed int) error

	AllRooms() ([]models.Room, error)
	GetRestrictionsForRoomByDate(roomID int, start, end time.Time) ([]models.RoomRestriction, error)

	InsertBlockForRoom(id int, startDate time.Time) error
	DeleteBlockByID(id int) error
}
