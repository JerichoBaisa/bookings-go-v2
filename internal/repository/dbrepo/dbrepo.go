package dbrepo

import (
	"database/sql"

	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/config"
	"bitbucket.org/JerichoBaisa/bookings-go-v2/internal/repository"
)

// make a repository itself
type postgresDBRepo struct {
	App *config.AppConfig
	DB  *sql.DB
}

type testDBRepo struct {
	App *config.AppConfig
	DB  *sql.DB
}

func NewPostgresRepo(conn *sql.DB, a *config.AppConfig) repository.DatabaseRepo {
	return &postgresDBRepo{
		App: a,
		DB:  conn,
	}
}

func NewTestingRepo(a *config.AppConfig) repository.DatabaseRepo {
	return &testDBRepo{
		App: a,
	}
}

// mysql
// type mySQLDBRepo struct {
// 	App *config.AppConfig
// 	DB  *sql.DB
// }

// func NewMySQLRepo(conn *sql.DB, a *config.AppConfig) repository.DatabaseRepo {
// 	return &mySQLDBRepo{
// 		App: a,
// 		DB:  conn,
// 	}
// }
