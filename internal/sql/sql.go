package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	codemail "ne-otchislyat/internal/codEmail"
	"sort"
	"time"

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
		verified BOOLEAN DEFAULT FALSE,
    	verification_code VARCHAR(6),
    	time_live_code TIMESTAMP,
		rating INT DEFAULT 0,
		tgUs VARCHAR(255) DEFAULT '',
		recvizits BIGINT DEFAULT 0,
		dateRegistr TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		countSdelanihZakazov INT DEFAULT 0
	);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS zakazs(
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
		user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name TEXT,
		title TEXT,
		discription TEXT,
		price INT NOT NULL,
		dateCreateCase TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS favorites(
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		card_id INT NOT NULL,
		card_type VARCHAR(10) NOT NULL CHECK (card_type IN ('zakaz', 'vakan')),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, card_id, card_type)  
	);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS comments(
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		card_id INT NOT NULL,
    	card_type VARCHAR(10) NOT NULL CHECK (card_type IN ('zakaz', 'vakan')),
		title TEXT,
		avtor TEXT,
		rating INT,
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
	var verified bool

	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		err = db.QueryRow("SELECT verified FROM users WHERE email = $1", email).Scan(&verified)
		if err != nil {
			return err
		}

		if verified {
			return errors.New("email exist")
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		code := codemail.GenerateCode()
		timeLiveCode := time.Now().Add(15 * time.Minute)

		_, err = db.Exec(`
			UPDATE users 
			SET password = $1, name = $2, verification_code = $3, time_live_code = $4
			WHERE email = $5`,
			string(hashedPassword), name, code, timeLiveCode, email)
		if err != nil {
			return err
		}

		go func() {
			err := codemail.SendVerificationCode(email, code)
			if err != nil {
				log.Printf("Failed to send email to %s: %v", email, err)
			}
		}()

		return errors.New("user not verified")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	code := codemail.GenerateCode()
	timeLiveCode := time.Now().Add(15 * time.Minute)

	_, err = db.Exec(`
		INSERT INTO users(email, password, name, verified, verification_code, time_live_code) 
		VALUES ($1, $2, $3, $4, $5, $6)`,
		email, string(hashedPassword), name, false, code, timeLiveCode)
	if err != nil {
		return err
	}

	go func() {
		err := codemail.SendVerificationCode(email, code)
		if err != nil {
			log.Printf("Failed to send email to %s: %v", email, err)
		}
	}()

	return nil
}
func UpdateUserVerified(email string) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Fail update verify", err)
		return err
	}

	defer db.Close()

	_, err = db.Exec(`UPDATE users SET verified = true, verification_code = NULL WHERE email = $1`, email)
	if err != nil {
		log.Fatal("Fail update verify", err)
		return err
	}
	return nil
}
func VerifyCodeInSql(email string) (string, time.Time, bool, error) {
	var storedCode string
	var expires time.Time
	var verified bool

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Fail open Db", err)
		return "", time.Time{}, false, err
	}
	defer db.Close()

	err = db.QueryRow(`
		SELECT verification_code, time_live_code, verified 
		FROM users WHERE email = $1
	`, email).Scan(&storedCode, &expires, &verified)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", time.Time{}, false, errors.New("user not found")
		}
		return "", time.Time{}, false, err
	}

	return storedCode, expires, verified, nil
}
func LoginDb(email, password string) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Fail open Db", err)
		return err
	}
	defer db.Close()

	var storedHash string
	var verified bool
	var userID int

	err = db.QueryRow(`
		SELECT id, password, verified 
		FROM users 
		WHERE email = $1`, email).Scan(&userID, &storedHash, &verified)

	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("user not found")
		}
		return err
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		return errors.New("wrong password")
	}

	if !verified {
		code := codemail.GenerateCode()
		timeLiveCode := time.Now().Add(15 * time.Minute)

		_, err = db.Exec(`
			UPDATE users 
			SET verification_code = $1, time_live_code = $2 
			WHERE id = $3`,
			code, timeLiveCode, userID)
		if err != nil {
			log.Printf("Failed to update verification code: %v", err)
		}

		go func() {
			err := codemail.SendVerificationCode(email, code)
			if err != nil {
				log.Printf("Failed to send email to %s: %v", email, err)
			}
		}()

		return errors.New("email not verified")
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
	Id                      int       `json:"id"`
	Password                string    `json:"password"`
	Name                    string    `json:"name"`
	Email                   string    `json:"email"`
	Rating                  int       `json:"rating"`
	TgUs                    string    `json:"tgUs"`
	Recvizits               int64     `json:"recvivits"`
	DateCreateprofileStruct string    `json:"dateCreateprofileStruct"`
	CountSdelanihZakazov    int       `json:"countSdelanihZakazov:`
	Cards                   []Cards   `json:"cards"`
	Comments                []comment `json:"comments"`
}

func GetInfProfile(email string) (profileStruct, error) {
	var InfprofileStruct profileStruct

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Fail open Db", err)
		return profileStruct{}, err
	}
	defer db.Close()

	rowsUser, err := db.Query("SELECT id, password, name, rating, tgUs, recvizits, dateRegistr, countSdelanihZakazov FROM users WHERE email = $1", email)
	if err != nil {
		return profileStruct{}, err
	}
	defer rowsUser.Close()

	InfprofileStruct.Email = email
	var userId int
	for rowsUser.Next() {
		err := rowsUser.Scan(&userId, &InfprofileStruct.Password, &InfprofileStruct.Name, &InfprofileStruct.Rating, &InfprofileStruct.TgUs, &InfprofileStruct.Recvizits, &InfprofileStruct.DateCreateprofileStruct, &InfprofileStruct.CountSdelanihZakazov)
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

	var itemID int
	query := fmt.Sprintf(`INSERT INTO %s (user_id, name, title, discription, price) VALUES ($1, $2, $3, $4, $5) RETURNING id`, tableName)
	err = tx.QueryRow(query, userID, name, title, discription, price).Scan(&itemID)
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
