package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	_ "github.com/lib/pq"
)

const connStr = "host=localhost port=5432 user=root password=Password123! dbname=neotchislyat sslmode=disable"

func InitDB() error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users(
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		name VARCHAR(255) DEFAULT 'User',
		isCompany BOOLEAN DEFAULT FALSE,
		rating INT DEFAULT 0,
		tgUs VARCHAR(255) DEFAULT '',
		recvizits BIGINT DEFAULT 0,
		dateCreateprofileStruct TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS zakazs(
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL,
		name TEXT,
		title TEXT,
		discription TEXT,
		price INT NOT NULL,
		dateCreateCase TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS vakans(
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL,
		name TEXT,
		title TEXT,
		discription TEXT,
		price INT NOT NULL,
		dateCreateCase TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS comments(
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL,
		title TEXT,
		avtor TEXT,
		rating INT CHECK (rating >= 1 AND rating <= 5),
		dateCreateComment TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tags(
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) UNIQUE NOT NULL,
		category VARCHAR(50) DEFAULT 'other'
	);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS zakaz_tags(
		zakaz_id INT REFERENCES zakazs(id) ON DELETE CASCADE,
		tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
		PRIMARY KEY (zakaz_id, tag_id)
	);`)
	if err != nil {
		return err
	}

	return nil
}

func RegDb(email, password, name string) error {
	if len(name) < 1 {
		name = "User"
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		InitDB()
	}

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Fail open Db", err)
		return err
	}
	defer db.Close()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("email exist")
	}

	_, err = db.Exec("INSERT INTO users(email, password, name) VALUES ($1, $2, $3)", email, password, name)
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

type profileStruct struct {
	Name                    string    `json:"name"`
	Email                   string    `json:"email"`
	IsCompany               bool      `json:"isCompany"`
	Rating                  int       `json:"rating"`
	TgUs                    string    `json:"tgUs"`
	Recvizits               int64     `json:"recvivits"`
	Cases                   []cases   `json:"cases"`
	Comments                []comment `json:"comments"`
	DateCreateprofileStruct string    `json:"dateCreateprofileStruct"`
}

func GetInfProfile(email string) (profileStruct, error) {
	var InfprofileStruct profileStruct
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Fail open Db", err)
		return profileStruct{}, err
	}
	defer db.Close()

	rowsUser, err := db.Query("SELECT id, name, isCompany, rating, tgUs, recvizits, dateCreateprofileStruct FROM users WHERE email = $1", email)
	if err != nil {
		return profileStruct{}, err
	}
	defer rowsUser.Close()

	InfprofileStruct.Email = email
	var userId int
	for rowsUser.Next() {
		err := rowsUser.Scan(&userId, &InfprofileStruct.Name, &InfprofileStruct.IsCompany, &InfprofileStruct.Rating, &InfprofileStruct.TgUs, &InfprofileStruct.Recvizits, &InfprofileStruct.DateCreateprofileStruct)
		if err != nil {
			return profileStruct{}, err
		}
	}

	rowsCases, err := db.Query("SELECT title, discription, price, dateCreateCase FROM cases WHERE user_id = $1", userId)
	if err != nil {
		return profileStruct{}, err
	}
	defer rowsCases.Close()

	var casesList []cases
	for rowsCases.Next() {
		var c cases
		err := rowsCases.Scan(&c.Title, &c.Discription, &c.Price, &c.DateCreateCase)
		if err != nil {
			return profileStruct{}, err
		}
		casesList = append(casesList, c)
	}
	InfprofileStruct.Cases = casesList

	rowsComments, err := db.Query("SELECT title, rating, avtor, dateCreateComment FROM comments WHERE user_id = $1", userId)
	if err != nil {
		return profileStruct{}, err
	}
	defer rowsComments.Close()

	var commentsList []comment
	for rowsComments.Next() {
		var com comment
		err := rowsComments.Scan(&com.Title, &com.Stars, &com.Avtor, &com.DateCreateComments)
		if err != nil {
			return profileStruct{}, err
		}
		commentsList = append(commentsList, com)
	}
	InfprofileStruct.Comments = commentsList

	return InfprofileStruct, nil
}

func UpdateProf(name string, password string, isCompany bool, Rating int, TgUs string, Recvizits int64, email string) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Fail open Db", err)
		return err
	}
	defer db.Close()
	_, err = db.Exec(`
		UPDATE users SET 
			name = $1,
			password = $2,
			isCompany = $3,
			rating = $4,
			tgUs = $5,
			recvizits = $6
		WHERE email = $7`,
		name, password, isCompany, Rating, TgUs, Recvizits, email)
	if err != nil {
		log.Println("Update error:", err)
		log.Fatal("Fail write inf of profile in Db", err)
		return err
	}
	return nil
}

type Zakaz struct {
	Label       string   `json:"label"`
	Discription string   `json:"discription"`
	Avtor       string   `json:"avtor"`
	Price       int      `json:"price"`
	Tags        []string `json:"tags"`
}

func AddZakaz(userID int, name, title, discription string, price int, tags []string) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var zakazID int
	err = tx.QueryRow(`
		INSERT INTO zakazs (user_id, name, title, discription, price)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, userID, name, title, discription, price).Scan(&zakazID)
	if err != nil {
		return err
	}

	for _, tagName := range tags {

		var tagID int
		err = tx.QueryRow(`
			INSERT INTO tags (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, tagName).Scan(&tagID)
		if err != nil {
			return err
		}

		// Связываем заказ с тегом
		_, err = tx.Exec(`
			INSERT INTO zakaz_tags (zakaz_id, tag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, zakazID, tagID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetCards возвращает заказы с пагинацией и фильтрацией по тегам
func GetCards(page int, tagFilters []string) ([]Zakaz, error) {
	var cards []Zakaz
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * 20

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Println("Fail open Db:", err)
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT z.id, z.name, z.title, z.discription, z.price,
		       COALESCE(array_agg(t.name) FILTER (WHERE t.name IS NOT NULL), '{}') as tags
		FROM zakazs z
		LEFT JOIN zakaz_tags zt ON z.id = zt.zakaz_id
		LEFT JOIN tags t ON zt.tag_id = t.id
		WHERE 1=1
	`
	args := []interface{}{}
	argCounter := 1

	// Фильтрация по тегам
	if len(tagFilters) > 0 {
		placeholders := make([]string, len(tagFilters))
		for i := range tagFilters {
			placeholders[i] = fmt.Sprintf("$%d", argCounter)
			args = append(args, tagFilters[i])
			argCounter++
		}
		query += fmt.Sprintf(`
			AND z.id IN (
				SELECT zakaz_id FROM zakaz_tags
				WHERE tag_id IN (
					SELECT id FROM tags WHERE name IN (%s)
				)
				GROUP BY zakaz_id
				HAVING COUNT(DISTINCT tag_id) = %d
			)
		`, strings.Join(placeholders, ","), len(tagFilters))
	}

	query += ` GROUP BY z.id ORDER BY z.id LIMIT 20 OFFSET $` + fmt.Sprint(argCounter)
	args = append(args, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Println("Fail query zakazs:", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var z Zakaz
		var tagsArray []string
		err := rows.Scan(&z.Avtor, &z.Label, &z.Discription, &z.Price, &tagsArray)
		if err != nil {
			log.Println("Scan error:", err)
			continue
		}
		z.Tags = tagsArray
		cards = append(cards, z)
	}

	return cards, nil
}
