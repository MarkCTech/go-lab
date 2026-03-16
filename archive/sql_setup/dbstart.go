package sql_db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Database struct {
	Db *sql.DB
	// Default Log In
	Username string
	Password string
	Hostname string
	Dbname   string
}

func MakeDbLogin(dbname string) *Database {
	// Set login details here
	dbstruct := new(Database)
	dbstruct.Username = "root"
	dbstruct.Password = "toor"
	dbstruct.Hostname = "127.0.0.1:3306"
	dbstruct.Dbname = dbname
	return dbstruct
}

func Dbstart(dbname string) *Database {
	// Logs in and sets up a sql database
	db, err := sql.Open("mysql", Dsn(""))
	if err != nil {
		log.Printf("Error %s when opening DB\n", err)
	}
	database := MakeDbLogin(dbname)

	ctx, cancelfunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	res, err := db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+dbname)
	if err != nil {
		log.Printf("Error %s when creating DB\n", err)
	}
	no, err := res.RowsAffected()
	if err != nil {
		log.Printf("Error %s when fetching rows", err)

	}
	log.Printf("rows affected %d\n", no)

	db.Close()
	db, err = sql.Open("mysql", Dsn(dbname))
	if err != nil {
		log.Printf("Error %s when opening DB", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(time.Minute * 5)
	ctx, cancelfunc = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelfunc()
	err = db.PingContext(ctx)
	if err != nil {
		log.Printf("Errors %s pinging DB", err)
	}
	log.Printf("Connected to DB %s successfully\n", dbname)
	CreateTable(db, "tasks")

	// Assign *sql.DB to the Database object
	database.Db = db
	return database
}

func CreateTable(db *sql.DB, tableName string) {
	db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %v", tableName))
	db.Exec(fmt.Sprintf("CREATE TABLE %v (id INT NOT NULL AUTO_INCREMENT PRIMARY KEY, title VARCHAR(100) NOT NULL, completed BOOLEAN NOT NULL DEFAULT 0)", tableName))
	db.Query(fmt.Sprintf("INSERT INTO %v (title) VALUES ('test')", tableName))
}

func Dsn(name string) string {
	d := MakeDbLogin(name)
	return fmt.Sprintf("%s:%s@tcp(%s)/%s", d.Username, d.Password, d.Hostname, name)
}
