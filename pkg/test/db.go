package test

import (
	"database/sql"
	"github.com/google/uuid"
	sqliteGo "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const (
	driverName = "sqlite3_extended"
	dsn        = ":memory:"
)

func newUUID(arguments ...interface{}) (string, error) {
	u, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func GetTestDB() *gorm.DB {
	if !stringInSlice(driverName, sql.Drivers()) {
		sql.Register(driverName,
			&sqliteGo.SQLiteDriver{
				ConnectHook: func(conn *sqliteGo.SQLiteConn) error {
					return conn.RegisterFunc("uuid_generate_v4", newUUID, true)
				},
			},
		)
	}

	conn, _ := sql.Open(driverName, dsn)
	db, _ := gorm.Open(sqlite.Dialector{
		DriverName: driverName,
		DSN:        dsn,
		Conn:       conn,
	})

	return db
}
