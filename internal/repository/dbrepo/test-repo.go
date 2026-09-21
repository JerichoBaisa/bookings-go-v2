package dbrepo

import (
	"errors"
	"log"
	"time"

	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/models"
)

func (m *testDBRepo) AllUsers() bool {
	return true
}

// InsertReservation insert a reservation into the database
func (m *testDBRepo) InsertReservation(res models.Reservation) (int, error) {
	if res.RoomID == 2 {
		return 0, errors.New("some error insertreservation")
	}
	return 1, nil
}

func (m *testDBRepo) InsertRoomRestriction(r models.RoomRestriction) error {
	if r.RoomID == 22 {
		return errors.New("some error insert room restriction")
	}
	return nil
}

// SearchAvailabilityByDatesByRoomID returntrue if exist and false if no availability
func (m *testDBRepo) SearchAvailabilityByDatesByRoomID(start, end time.Time, roomID int) (bool, error) {
	if roomID == 22 {
		return true, errors.New("some error insert room restriction")
	}
	return false, nil

}

func (m *testDBRepo) SearchAvailabilityForAllRooms(start, end time.Time) ([]models.Room, error) {
	// if the start date is after 2049-12-31, then return empty slice,
	// indicating no rooms are available;
	layout := "2006-01-02"
	str := "2049-12-31"
	t, err := time.Parse(layout, str)
	if err != nil {
		log.Println(err)
	}

	var rooms []models.Room
	testDateToFail, err := time.Parse(layout, "2060-01-01")
	if err != nil {
		log.Println(err)
	}

	if start == testDateToFail {
		return rooms, errors.New("some error")
	}

	if start.After(t) {
		return rooms, nil
	}

	room := models.Room{
		ID:       1,
		RoomName: "Generals Quarter",
	}

	rooms = append(rooms, room)

	return rooms, nil
}

// GetRoomByID( get room by id
func (m *testDBRepo) GetRoomByID(id int) (models.Room, error) {
	var room models.Room
	if id > 2 {
		return room, errors.New("some error get room id")
	}
	return room, nil
}

// USERS
func (m *testDBRepo) GetUserByID(id int) (models.User, error) {
	mu := models.User{}
	return mu, nil

}

func (m *testDBRepo) UpdateUser(u models.User) error {
	return nil
}

func (m *testDBRepo) Authenticate(email, testPassword string) (int, string, error) {
	if email == "invalid@gmail.com" {
		return 0, "invalid email", errors.New("some error in Authenticate")
	}
	return 1, "", nil
}

func (m *testDBRepo) GetAdminAllReservations() ([]models.Reservation, error) {
	var res []models.Reservation
	return res, nil
}

func (m *testDBRepo) GetAdminAllNewReservations() ([]models.Reservation, error) {
	var res []models.Reservation
	return res, nil
}

func (m *testDBRepo) GetReservationByID(id int) (models.Reservation, error) {
	var res models.Reservation
	if id == 200 {
		return res, errors.New("some error GetReservationByID")
	}
	return res, nil
}

func (m *testDBRepo) UpdateReservation(u models.Reservation, id int) error {
	if id == 200 {
		return errors.New("some error UpdateReservation")
	}
	return nil
}

func (m *testDBRepo) DeleteReservation(id int) error {
	if id == 200 {
		return errors.New("some error DeleteReservation")
	}
	return nil
}

// UpdateProcessedForReservation update processed for a reservation by id
func (m *testDBRepo) UpdateProcessedForReservation(id, processed int) error {
	if id == 200 && processed == 200 {
		return errors.New("some error UpdateProcessedForReservation")
	}
	return nil
}

func (m *testDBRepo) AllRooms() ([]models.Room, error) {
	var rooms []models.Room
	return rooms, nil
}

func (m *testDBRepo) GetRestrictionsForRoomByDate(roomID int, start, end time.Time) ([]models.RoomRestriction, error) {
	var rr []models.RoomRestriction
	return rr, nil
}

// InsertBlockForRoom insert a room restriction
func (m *testDBRepo) InsertBlockForRoom(id int, startDate time.Time) error {
	if id == 200 {
		return errors.New("some error InsertBlockForRoom")
	}
	return nil
}

// DeleteBlockBiID deletes a room restriction
func (m *testDBRepo) DeleteBlockByID(id int) error {
	if id == 200 {
		return errors.New("some error DeleteBlockByID")
	}
	return nil
}
