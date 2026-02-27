package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const connStr = "host=localhost port=5432 user=postgres password=postgres dbname=neotchislyat sslmode=disable"

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
		name VARCHAR(255) UNIQUE NOT NULL,
		category VARCHAR(255) DEFAULT 'other'
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
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS vakans_tags(
    vakans_id INT REFERENCES vakans(id) ON DELETE CASCADE,
    tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (vakans_id, tag_id)
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
		log.Fatal("Fail open Db", err)
		return err
	}
	defer db.Close()

	err = InitDB()
	if err != nil {
		log.Fatal("Fail init Db", err)
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
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT INTO users(email, password, name) VALUES ($1, $2, $3)",
		email, string(hashedPassword), name)
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

	var storedHash string
	err = db.QueryRow("SELECT password FROM users WHERE email = $1", email).Scan(&storedHash)
	if err != nil {
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		return errors.New("wrong password")
	}
	return nil
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
	Cards                   []Cards   `json:"cards"`
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
	if userId == 0 {
		return profileStruct{}, errors.New("user not found")
	}

	rowsCards, err := db.Query("SELECT title, discription, price, dateCreateCase FROM zakazs WHERE user_id = $1", userId)
	if err != nil {
		return profileStruct{}, err
	}
	defer rowsCards.Close()

	var cardsList []Cards
	for rowsCards.Next() {
		var c Cards
		var date string
		err := rowsCards.Scan(&c.Label, &c.Discription, &c.Price, &date)
		c.Avtor = InfprofileStruct.Name

		if err != nil {
			return profileStruct{}, err
		}
		cardsList = append(cardsList, c)
	}
	InfprofileStruct.Cards = cardsList

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
		return err
	}
	return nil
}

type Cards struct {
	Label       string   `json:"label"`
	Discription string   `json:"discription"`
	Avtor       string   `json:"avtor"`
	Price       int      `json:"price"`
	Tags        []string `json:"tags"`
}

func AddItem(tableName, email, name, title, discription string, price int, tags []string) error {
	tagsTable := ""
	tagsColumn := ""
	if tableName == "zakazs" {
		tagsTable = "zakaz_tags"
		tagsColumn = "zakaz_id"
	} else if tableName == "vakans" {
		tagsTable = "vakans_tags"
		tagsColumn = "vakans_id"
	} else {
		return errors.New("неверное поле tableName")
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		return errors.New("user not found")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Вставляем элемент
	var itemID int
	query := fmt.Sprintf(`INSERT INTO %s (user_id, name, title, discription, price) VALUES ($1, $2, $3, $4, $5) RETURNING id`, tableName)
	err = tx.QueryRow(query, userID, name, title, discription, price).Scan(&itemID)
	if err != nil {
		return err
	}

	for _, tagName := range tags {
		// Вставляем тег, если его нет
		var tagID int
		err = tx.QueryRow(`
			INSERT INTO tags (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, tagName).Scan(&tagID)
		if err != nil {
			return err
		}

		// Связываем элемент с тегом
		tagQuery := fmt.Sprintf(`INSERT INTO %s (%s, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, tagsTable, tagsColumn)
		_, err = tx.Exec(tagQuery, itemID, tagID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// возвращает слайс карточек пагинацией и фильтрацией по тегам
func GetCards(page int, tagFilters []string, itemType string, priceUpDownFalse string) ([]Cards, error) {
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

	// получаем ID элементов по тегам
	var itemIDs []int
	if len(tagFilters) > 0 {
		itemIDs, err = filterByTags(db, tagFilters, itemType)
		if err != nil {
			return nil, err
		}
		if len(itemIDs) == 0 {
			return []Cards{}, nil
		}
	}

	return getItemsByIDs(db, itemIDs, offset, itemType, priceUpDownFalse)
}

func filterByTags(db *sql.DB, tags []string, itemType string) ([]int, error) {
	var tagsTable string
	var idColumn string

	if itemType == "zakazs" {
		tagsTable = "zakaz_tags"
		idColumn = "zakaz_id"
	} else if itemType == "vakans" {
		tagsTable = "vakans_tags"
		idColumn = "vakans_id"
	} else {
		return nil, errors.New("неверный тип элемента")
	}

	rows, err := db.Query(fmt.Sprintf(`SELECT %s FROM %s WHERE tag_id IN (SELECT id FROM tags WHERE name = ANY($1)) GROUP BY %s HAVING COUNT(DISTINCT tag_id) = $2 `, idColumn, tagsTable, idColumn), pq.Array(tags), len(tags))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func getItemsByIDs(db *sql.DB, ids []int, offset int, itemType string, priceUpDownFalse string) ([]Cards, error) {
	var tableName, tagsTable, idColumn string

	if itemType == "zakazs" {
		tableName = "zakazs"
		tagsTable = "zakaz_tags"
		idColumn = "zakaz_id"
	} else if itemType == "vakans" {
		tableName = "vakans"
		tagsTable = "vakans_tags"
		idColumn = "vakans_id"
	} else {
		return nil, errors.New("неверный тип элемента")
	}

	query := fmt.Sprintf(`
		SELECT %s.id, %s.name, %s.title, %s.discription, %s.price,
		COALESCE(array_agg(tags.name) FILTER (WHERE tags.name IS NOT NULL), '{}') as tags
		FROM %s
		LEFT JOIN %s ON %s.id = %s.%s
		LEFT JOIN tags ON %s.tag_id = tags.id
	`, tableName, tableName, tableName, tableName, tableName,
		tableName, tagsTable, tableName, tagsTable, idColumn, tagsTable)

	args := []interface{}{}

	if len(ids) > 0 {
		query += ` WHERE ` + tableName + `.id = ANY($1)`
		args = append(args, pq.Array(ids))
	}

	query += ` GROUP BY ` + tableName + `.id ORDER BY ` + tableName + `.id LIMIT 20 OFFSET $` + fmt.Sprint(len(args)+1)
	args = append(args, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []Cards
	for rows.Next() {
		var z Cards
		var tagsArray []string
		var id int
		var name, title, discription string
		var price int

		err := rows.Scan(&id, &name, &title, &discription, &price, pq.Array(&tagsArray))
		if err != nil {
			log.Println("Scan error:", err)
			continue
		}

		z.Avtor = name
		z.Label = title
		z.Discription = discription
		z.Price = price
		z.Tags = tagsArray

		cards = append(cards, z)
	}

	if priceUpDownFalse == "Up" {
		sort.Slice(cards, func(i, j int) bool {
			return cards[i].Price < cards[j].Price
		})
	} else if priceUpDownFalse == "Down" {
		sort.Slice(cards, func(i, j int) bool {
			return cards[i].Price > cards[j].Price
		})
	}
	if cards == nil {
		return []Cards{}, nil
	}

	return cards, nil
}
