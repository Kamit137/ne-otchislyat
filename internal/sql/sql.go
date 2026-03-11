package sql

//
import (
	"database/sql"
	"errors"
	"log"
	"ne-otchislyat/internal/codemail"
	"time"

	_ "github.com/lib/pq"
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
		balance BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
		frozen_balance BIGINT NOT NULL DEFAULT 0 CHECK (frozen_balance >= 0),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		countSdelanihZakazov INT DEFAULT 0
	);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS vakans(
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		avtor TEXT,
		title TEXT,
		discription TEXT,
		price INT NOT NULL,
		tag TEXT,
		dateCreateVakans TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS favorites(
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		vakans_id INT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, vakans_id)
	);`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS comments(
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		vakans_id INT NOT NULL,
		title TEXT,
		avtor TEXT,
		rating INT NOT NULL DEFAULT 5 CHECK (rating >= 1 AND rating <= 5),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
			WHERE email = $5`, string(hashedPassword), name, code, timeLiveCode, email)
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
		FROM users WHERE email = $1`, email).Scan(&storedCode, &expires, &verified)

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
			WHERE id = $3`, code, timeLiveCode, userID)
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

type Vakans struct {
	Label       string `json:"label"`
	Discription string `json:"discription"`
	Avtor       string `json:"avtor"`
	Price       int    `json:"price"`
	Tag         string `json:"tag"`
}

type profileStruct struct {
	Id                      int       `json:"id"`
	Password                string    `json:"password"`
	Name                    string    `json:"name"`
	Email                   string    `json:"email"`
	Rating                  int       `json:"rating"`
	TgUs                    string    `json:"tgUs"`
	Recvizits               int64     `json:"recvizits"`
	DateCreateprofileStruct string    `json:"dateCreateprofileStruct"`
	CountSdelanihZakazov    int       `json:"countSdelanihZakazov"`
	Vakans                  []Vakans  `json:"vakans"`
	Comments                []comment `json:"comments"`
}

func GetInfProfile(email string) (profileStruct, error) {
	var prof profileStruct

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Fail open Db", err)
		return profileStruct{}, err
	}
	defer db.Close()

	var userId int
	err = db.QueryRow(`
		SELECT id, password, name, rating, tgUs, balance, created_at, countSdelanihZakazov
		FROM users WHERE email = $1`, email).Scan(&userId, &prof.Password, &prof.Name, &prof.Rating,
		&prof.TgUs, &prof.Recvizits, &prof.DateCreateprofileStruct, &prof.CountSdelanihZakazov)
	if err != nil {
		if err == sql.ErrNoRows {
			return profileStruct{}, errors.New("user not found")
		}
		return profileStruct{}, err
	}
	prof.Email = email
	prof.Id = userId

	rowsVakans, err := db.Query(`
		SELECT title, discription, price, tag
		FROM vakans WHERE user_id = $1`, userId)
	if err != nil {
		return profileStruct{}, err
	}
	defer rowsVakans.Close()

	var vakansList []Vakans
	for rowsVakans.Next() {
		var v Vakans
		err := rowsVakans.Scan(&v.Label, &v.Discription, &v.Price, &v.Tag)
		if err != nil {
			return profileStruct{}, err
		}
		v.Avtor = prof.Name
		vakansList = append(vakansList, v)
	}
	if vakansList == nil {
		vakansList = []Vakans{}
	}
	prof.Vakans = vakansList

	rowsComments, err := db.Query(`
		SELECT title, rating, avtor, created_at
		FROM comments WHERE user_id = $1`, userId)
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
	if commentsList == nil {
		commentsList = []comment{}
	}
	prof.Comments = commentsList

	return prof, nil
}

func UpdateProf(name string, password string, isCompany bool, rating int, tgUs string, recvizits int64, email string) error {
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
			rating = $3,
			tgUs = $4,
			balance = $5
		WHERE email = $6`,
		name, password, rating, tgUs, recvizits, email)
	if err != nil {
		log.Println("Update error:", err)
		return err
	}
	return nil
}

func AddVakans(email, avtor, title, discription, tag string, price int) error {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	var userID int
	err = db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("user not found")
		}
		return err
	}

	_, err = db.Exec(`
		INSERT INTO vakans (user_id, name, title, discription, price, tag)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, avtor, title, discription, price, tag)
	return err
}

func GetVakans(page int, tag, priceUpDownFalse string) ([]Vakans, error) {
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
	var rows *sql.Rows
	if tag != "" {
		switch priceUpDownFalse {
		case "Up":
			rows, err = db.Query(`SELECT name, title, discription, price, tag FROM vakans WHERE tag = $1 ORDER BY price ASC LIMIT 20 OFFSET $2`, tag, offset)
		case "Down":
			rows, err = db.Query(`SELECT name, title, discription, price, tag FROM vakans WHERE tag = $1 ORDER BY price DESC LIMIT 20 OFFSET $2`, tag, offset)
		default:
			rows, err = db.Query(`SELECT name, title, discription, price, tag FROM vakans WHERE tag = $1 ORDER BY id ASC LIMIT 20 OFFSET $2`, tag, offset)
		}
	} else {
		switch priceUpDownFalse {
		case "Up":
			rows, err = db.Query(`SELECT name, title, discription, price, tag FROM vakans ORDER BY price ASC LIMIT 20 OFFSET $1`, offset)
		case "Down":
			rows, err = db.Query(`SELECT name, title, discription, price, tag FROM vakans ORDER BY price DESC LIMIT 20 OFFSET $1`, offset)
		default:
			rows, err = db.Query(`SELECT name, title, discription, price, tag FROM vakans ORDER BY id ASC LIMIT 20 OFFSET $1`, offset)
		}
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vakansList []Vakans
	for rows.Next() {
		var v Vakans
		err := rows.Scan(&v.Avtor, &v.Label, &v.Discription, &v.Price, &v.Tag)
		if err != nil {
			return nil, err
		}
		vakansList = append(vakansList, v)
	}

	if vakansList == nil {
		return []Vakans{}, nil
	}
	return vakansList, nil
}
