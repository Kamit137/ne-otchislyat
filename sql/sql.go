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
    name TEXT DEFAULT 'User',
    isCompany BOOLEAN DEFAULT FALSE,
    rating INTEGER DEFAULT 0,
    tgUs TEXT DEFAULT '',        -- ← добавил
    recvizits BIGINT DEFAULT 0,  -- ← добавил
    dateCreateProfile TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`)
	if err != nil {
		log.Fatal("ER create db", err)
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS cases(
                id SERIAL PRIMARY KEY,
                user_id INTEGER NOT NULL,
                title TEXT,
				discription TEXT,
				price INTEGER NOT NULL,
				dateCreateCase TIMESTAMP DEFAULT CURRENT_TIMESTAMP);`)
	if err != nil {
		log.Fatal("ER create db", err)
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS comments(
                id SERIAL PRIMARY KEY,
                user_id INTEGER NOT NULL,
                title TEXT,
				avtor TEXT,
				rating INTEGER CHECK (rating >= 1 AND rating <= 5),
				dateCreateComment TIMESTAMP DEFAULT CURRENT_TIMESTAMP);`)
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

type cases struct {
	Title          string `json:"title"`
	Discription    string `json:"discription"`
	Price          int    `json:"price"`
	DateCreateCase string `json:"dateCreateCreat"`
}

type comment struct {
	Title              string `json:"title"`
	Stars              int    `json:"stars"`
	Avtor              string `json:"avtor"`
	DateCreateComments string `json:"dateCreateComments"`
}

type profile struct {
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	IsCompany         bool      `json:"isCompany"`
	Rating            int       `json:"rating"`
	TgUs              string    `json:"tgUs"`
	Recvizits         int64     `json:"recvivits"`
	Cases             []cases   `json:"cases"`
	Comments          []comment `json:"comments"`
	DateCreateProfile string    `json:"dateCreateProfile"`
}

func GetInfProfile(email string) (profile, error) {
	var InfProfile profile
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Fail open Db", err)
		return profile{}, err
	}
	defer db.Close()

	rowsUser, err := db.Query("SELECT id, name, isCompany, rating, tgUs, recvizits, dateCreateProfile FROM users WHERE email=$1", email)
	if err != nil {
		return profile{}, err
	}
	defer rowsUser.Close()

	InfProfile.Email = email
	var userId int
	for rowsUser.Next() {
		err := rowsUser.Scan(&userId, &InfProfile.Name, &InfProfile.IsCompany, &InfProfile.Rating, &InfProfile.TgUs, &InfProfile.Recvizits, &InfProfile.DateCreateProfile)
		if err != nil {
			return profile{}, err
		}
	}

	rowsCases, err := db.Query("SELECT title, discription, price, dateCreateCase FROM cases WHERE user_id=$1", userId)
	if err != nil {
		return profile{}, err
	}
	defer rowsCases.Close()

	var casesList []cases
	for rowsCases.Next() {
		var c cases
		err := rowsCases.Scan(&c.Title, &c.Discription, &c.Price, &c.DateCreateCase)
		if err != nil {
			return profile{}, err
		}
		casesList = append(casesList, c)
	}
	InfProfile.Cases = casesList

	rowsComments, err := db.Query("SELECT title, rating, avtor, dateCreateComment FROM comments WHERE user_id=$1", userId)
	if err != nil {
		return profile{}, err
	}
	defer rowsComments.Close()

	var commentsList []comment
	for rowsComments.Next() {
		var com comment
		err := rowsComments.Scan(&com.Title, &com.Stars, &com.Avtor, &com.DateCreateComments)
		if err != nil {
			return profile{}, err
		}
		commentsList = append(commentsList, com)
	}
	InfProfile.Comments = commentsList

	return InfProfile, nil
}
