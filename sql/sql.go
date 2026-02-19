package sql

import (
	"database/sql"
	"errors"
	"log"

	_ "github.com/lib/pq"
)

const connStr = "host=localhost port=5432 user=postgres password=postgres dbname=neotchislyat sslmode=disable"

func RegDb(email, password string) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Fail open Db", err)
		return err
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users(
                id SERIAL PRIMARY KEY,
                email TEXT UNIQUE NOT NULL,
                password TEXT NOT NULL,
                name TEXT NOT NULL,
				is_company BOOLEAN DEFAULT FALSE,
				rating INTEGER CHECK (rating >= 1 AND rating <= 5),
				tg_us TEXT,
				recvizits BIGINT,
                date_create TIMESTAMP DEFAULT CURRENT_TIMESTAMP);`)
	if err != nil {
		log.Fatal("ER create db", err)
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS case(
                id SERIAL PRIMARY KEY,
                user_id INTEGER NOT NULL,
                title TEXT,
				discription TEXT,
				price INTEGER NOT NUL,
				date_create TIMESTAMP DEFAULT CURRENT_TIMESTAMP;)
				;`)
	if err != nil {
		log.Fatal("ER create db", err)
		return err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS comments(
                id SERIAL PRIMARY KEY,
                user_id INTEGER NOT NULL,
                title TEXT,
				rating INTEGER CHECK (rating >= 1 AND rating <= 5),
				date_create TIMESTAMP DEFAULT CURRENT_TIMESTAMP;)
				;`)
	if err != nil {
		log.Fatal("ER create db", err)
		return err
	}
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return errors.New("email exist")
	}

	_, err = db.Exec("INSERT INTO users(email, password) VALUES ($1, $2)", email, password)
	if err != nil {
		return err
	}

	return nil
}

func LoginDb(email, password string) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Fail open Db", err)
		return err
	}
	defer db.Close()

	var storedPassword string
	err = db.QueryRow("SELECT password FROM users WHERE email = $1", email).Scan(&storedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("user not found")
		}
		return err
	}

	if storedPassword != password {
		return errors.New("wrong password")
	}

	return nil
}

type task struct {
	Text   string
	Status int
}

func GetTasks(email string) ([]task, error) {
	var tasks []task

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Fail open Db", err)
		return tasks, err
	}
	defer db.Close()

	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return tasks, errors.New("user not found")
		}
		log.Fatal("Fail find user_id", err)
		return tasks, err
	}

	rows, err := db.Query("SELECT task, status FROM tasks WHERE user_id = $1", userID)
	if err != nil {
		log.Fatal("Fail find tasks", err)
		return tasks, err
	}
	defer rows.Close()

	for rows.Next() {
		var t task
		err := rows.Scan(&t.Text, &t.Status)
		if err != nil {
			return tasks, err
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}
