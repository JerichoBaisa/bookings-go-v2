package dbrepo

import (
	"context"
	"errors"
	"log"
	"time"

	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func (m *postgresDBRepo) AllUsers() bool {
	return true
}

// InsertReservation insert a reservation into the database
func (m *postgresDBRepo) InsertReservation(res models.Reservation) (int, error) {

	// web application is unpredictable,
	// What happens in the mid of transaction the user disconnect or close the browser?
	// if something goes wrong we can use context here as function below that
	//if the transaction below not "insert" into the database within 3second(somethin wrong with our application) we can cancel
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// stmt := `INSERT INTO reservations (first_name, last_name, email, phone, start_date, end_date, room_id, created_at, updated_at)
	// 						   VALUES ($firstName, $lastName, $email, $phone, $startDate ,$endDate, $roomID, $createdAt, $updatedAt)`

	var newID int
	stmt := `INSERT INTO reservations (first_name, last_name, email, phone, start_date, end_date, room_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) returning id`
	// ExecContext this is more robust and safer
	err := m.DB.QueryRowContext(ctx, stmt,
		res.FirstName,
		res.LastName,
		res.Email,
		res.Phone,
		res.StartDate,
		res.EndDate,
		res.RoomID,
		time.Now(),
		time.Now(),
	).Scan(&newID)

	if err != nil {
		return 0, err
	}

	return newID, nil
}

func (m *postgresDBRepo) InsertRoomRestriction(r models.RoomRestriction) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `INSERT INTO room_restrictions (start_date, end_date, room_id, reservation_id, restriction_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := m.DB.ExecContext(ctx, stmt,
		r.StartDate,
		r.Enddate,
		r.RoomID,
		r.ReservationID,
		r.RestrictionID,
		r.CreatedAt,
		r.UpdatedAt,
	)

	if err != nil {
		return err
	}

	return nil
}

// SearchAvailabilityByDatesByRoomID returntrue if exist and false if no availability
func (m *postgresDBRepo) SearchAvailabilityByDatesByRoomID(start, end time.Time, roomID int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var numRows int

	query := `select
				count(id)
			from 
				room_restrictions rr
			where room_id = $1 
			and $2 < end_date and $3 > start_date;`

	row := m.DB.QueryRowContext(ctx, query, roomID, start, end)
	err := row.Scan(&numRows)
	if err != nil {
		return false, err
	}

	if numRows == 0 {
		return true, nil
	}

	return false, nil

}

func (m *postgresDBRepo) SearchAvailabilityForAllRooms(start, end time.Time) ([]models.Room, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var rooms []models.Room
	query := `select
	r.id,
	r.room_name
from
	rooms r
where
	r.id not in (
	select
		rr.room_id
	from
		room_restrictions rr
	where
		$1 < rr.end_date
		and $2 > rr.start_date );`

	rows, err := m.DB.QueryContext(ctx, query, start, end)
	if err != nil {
		return rooms, nil
	}

	for rows.Next() {
		var room models.Room

		rows.Scan(
			&room.ID,
			&room.RoomName,
		)

		if err != nil {
			return rooms, nil
		}

		rooms = append(rooms, room)
	}

	if err = rows.Err(); err != nil {
		return rooms, err
	}
	return rooms, nil
}

// GetRoomByID( get room by id
func (m *postgresDBRepo) GetRoomByID(id int) (models.Room, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var room models.Room

	query := `SELECT id, room_name, created_at, updated_at FROM rooms WHERE id = $1`

	row := m.DB.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&room.ID,
		&room.RoomName,
		&room.CreatedAt,
		&room.UpdatedAt,
	)

	if err != nil {
		return room, err
	}

	return room, nil
}

func (m *postgresDBRepo) GetUserByID(id int) (models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `SELEC id, first_name, last_name, email, password, access_level, created_at, updated_at 
				FROM users where id = $1`
	row := m.DB.QueryRowContext(ctx, query, id)

	var usr models.User
	err := row.Scan(
		&usr.ID,
		&usr.FirstName,
		&usr.LastName,
		&usr.Email,
		&usr.Password,
		&usr.AccessLevel,
		&usr.CreatedAt,
		&usr.UpdatedAt,
	)

	if err != nil {
		return usr, err
	}

	return usr, nil
}

// UpdateUser update users in the database
func (m *postgresDBRepo) UpdateUser(u models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query := `UPDATE users SET first_name = $1, last_name = $2, email = $3, access_level = $4, updated_at = $5 `

	_, err := m.DB.ExecContext(ctx, query,
		u.FirstName,
		u.LastName,
		u.Email,
		u.AccessLevel,
		time.Now())

	if err != nil {
		return err
	}

	return nil
}

// Authenticate Authenticate a users
func (m *postgresDBRepo) Authenticate(email, testPassword string) (int, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	var hashedPassword string

	query := `SELECT id, password FROM users where email = $1`

	row := m.DB.QueryRowContext(ctx, query, email)
	err := row.Scan(&id, &hashedPassword)
	if err != nil {
		return id, "Invalid login creadentials", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(testPassword))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return 0, "Incorrect password", errors.New("incorrect password")
	} else if err != nil {
		return 0, "", err
	}

	return id, hashedPassword, nil

}

// Admin
// GetAdminAllReservations returns a slice of all reservations
func (m *postgresDBRepo) GetAdminAllReservations() ([]models.Reservation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var reservation []models.Reservation

	query := `SELECT r.id, r.first_name, r.last_name, r.email, r.phone, 
			r.start_date, r.end_date, r.room_id, r.created_at, r.updated_at,
	 		rm.id, rm.room_name, r.processed
			FROM reservations AS r 
			LEFT JOIN rooms AS rm ON (r.room_id = rm.id)
			ORDER BY r.start_date ASC`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return reservation, err
	}

	defer rows.Close()

	for rows.Next() {
		var i models.Reservation

		err := rows.Scan(
			&i.ID,
			&i.FirstName,
			&i.LastName,
			&i.Email,
			&i.Phone,
			&i.StartDate,
			&i.EndDate,
			&i.RoomID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Room.ID,
			&i.Room.RoomName,
			&i.Processed,
		)

		if err != nil {
			return reservation, err
		}

		reservation = append(reservation, i)
	}

	if err = rows.Err(); err != nil {
		return reservation, err
	}

	return reservation, nil
}

func (m *postgresDBRepo) GetAdminAllNewReservations() ([]models.Reservation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var reservation []models.Reservation

	query := `SELECT r.id, r.first_name, r.last_name, r.email, r.phone, 
			r.start_date, r.end_date, r.room_id, r.created_at, r.updated_at,
	 		rm.id, rm.room_name, r.processed 
			FROM reservations AS r 
			LEFT JOIN rooms AS rm ON (r.room_id = rm.id)
			WHERE r.processed = 0
			ORDER BY r.start_date ASC`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return reservation, err
	}

	defer rows.Close()

	for rows.Next() {
		var i models.Reservation

		err := rows.Scan(
			&i.ID,
			&i.FirstName,
			&i.LastName,
			&i.Email,
			&i.Phone,
			&i.StartDate,
			&i.EndDate,
			&i.RoomID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Room.ID,
			&i.Room.RoomName,
			&i.Processed,
		)

		if err != nil {
			return reservation, err
		}

		reservation = append(reservation, i)
	}

	if err = rows.Err(); err != nil {
		return reservation, err
	}

	return reservation, nil
}

func (m *postgresDBRepo) GetReservationByID(id int) (models.Reservation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var res models.Reservation

	query := `SELECT r.id, r.first_name, r.Last_name, r.email, r.phone, r.start_date, r.end_date, 
		r.created_at, r.updated_at, r.processed, rm.id, rm.room_name
		FROM reservations AS r LEFT JOIN rooms AS rm ON (r.room_id = rm.id)
		WHERE r.id = $1`

	row := m.DB.QueryRowContext(ctx, query, id)

	err := row.Scan(
		&res.ID,
		&res.FirstName,
		&res.LastName,
		&res.Email,
		&res.Phone,
		&res.StartDate,
		&res.EndDate,
		&res.CreatedAt,
		&res.UpdatedAt,
		&res.Processed,
		&res.Room.ID,
		&res.Room.RoomName,
	)
	if err != nil {
		return res, err
	}

	if err = row.Err(); err != nil {
		return res, err
	}

	return res, nil

}

// UpdateReservation update reservation in the database
func (m *postgresDBRepo) UpdateReservation(u models.Reservation, id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query := `UPDATE reservations SET first_name = $1, last_name = $2, email = $3, phone = $4, updated_at = $5 WHERE id = $6 `

	_, err := m.DB.ExecContext(ctx, query,
		u.FirstName,
		u.LastName,
		u.Email,
		u.Phone,
		time.Now(), id)

	if err != nil {
		return err
	}

	return nil
}

// DeleteReservation delete one reservation by id
func (m *postgresDBRepo) DeleteReservation(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `DELETE FROM reservations WHERE id = $1 `
	_, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	return nil

}

// UpdateProcessedForReservation update processed for a reservation by id
func (m *postgresDBRepo) UpdateProcessedForReservation(id, processed int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `UPDATE reservations SET processed = $1 WHERE id = $2 `
	_, err := m.DB.ExecContext(ctx, query, processed, id)
	if err != nil {
		return err
	}

	return nil
}

func (m *postgresDBRepo) AllRooms() ([]models.Room, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var rooms []models.Room

	query := `SELECT id, room_name, created_at, updated_at FROM rooms ORDER BY room_name ASC`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return rooms, err
	}

	defer rows.Close()

	for rows.Next() {
		var rm models.Room

		err := rows.Scan(
			&rm.ID,
			&rm.RoomName,
			&rm.CreatedAt,
			&rm.UpdatedAt,
		)

		if err != nil {
			return rooms, err
		}

		rooms = append(rooms, rm)
	}

	if err = rows.Err(); err != nil {
		return rooms, err
	}

	return rooms, nil
}

func (m *postgresDBRepo) GetRestrictionsForRoomByDate(roomID int, start, end time.Time) ([]models.RoomRestriction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var restrictions []models.RoomRestriction

	query := `SELECT id, coalesce(reservation_id, 0) as reservation_id, restriction_id, room_id, start_date, end_date
				FROM room_restrictions WHERE $1 < end_date AND $2 >= start_date
				AND room_id = $3`

	rows, err := m.DB.QueryContext(ctx, query, start, end, roomID)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var res models.RoomRestriction
		err := rows.Scan(
			&res.ID,
			&res.ReservationID,
			&res.RestrictionID,
			&res.RoomID,
			&res.StartDate,
			&res.Enddate,
		)

		if err != nil {
			return restrictions, err
		}

		restrictions = append(restrictions, res)
	}

	if err = rows.Err(); err != nil {
		return restrictions, err
	}

	return restrictions, nil

}

// InsertBlockForRoom insert a room restriction
func (m *postgresDBRepo) InsertBlockForRoom(id int, startDate time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `INSERT INTO room_restrictions (start_date, end_date, room_id, restriction_id, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := m.DB.ExecContext(ctx, query, startDate, startDate.AddDate(0, 0, 1), id, 2, time.Now(), time.Now())
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// DeleteBlockBiID deletes a room restriction
func (m *postgresDBRepo) DeleteBlockByID(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `DELETE FROM room_restrictions WHERE id = $1`
	_, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
