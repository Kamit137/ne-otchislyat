package sql

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

var DB *sql.DB

func InitDB() error {
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(25)
	DB.SetConnMaxLifetime(5 * time.Minute)

	err = DB.Ping()
	if err != nil {
		return err
	}

	_, err = DB.Exec(`CREATE TABLE IF NOT EXISTS users(
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
		recvizits BIGINT NOT NULL DEFAULT 0,
		frozen_balance BIGINT NOT NULL DEFAULT 0 CHECK (frozen_balance >= 0),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		countSdelanihZakazov INT DEFAULT 0
	);`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`CREATE TABLE IF NOT EXISTS vakans(
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

	_, err = DB.Exec(`CREATE TABLE IF NOT EXISTS favorites(
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		vakans_id INT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(user_id, vakans_id)
	);`)
	if err != nil {
		return err
	}

	_, err = DB.Exec(`CREATE TABLE IF NOT EXISTS comments(
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
	_, err = DB.Exec(`CREATE TABLE IF NOT EXISTS transactions (
		id BIGSERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
		order_id BIGINT,
		type TEXT NOT NULL CHECK (type IN ('deposit', 'freeze', 'unfreeze', 'payment', 'refund')),
		amount BIGINT NOT NULL CHECK (amount > 0),
		status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'success', 'failed')),
		payment_id TEXT UNIQUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`)
	if err != nil {
		return err
	}
	_, err = DB.Exec(`CREATE TABLE IF NOT EXISTS orders (
		id BIGSERIAL PRIMARY KEY,
		vakans_id INT REFERENCES vakans(id) ON DELETE SET NULL,
		client_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
		executor_id INT REFERENCES users(id) ON DELETE SET NULL,
		title TEXT NOT NULL,
		description TEXT,
		price BIGINT NOT NULL CHECK (price > 0),
		status TEXT NOT NULL DEFAULT 'pending',
		
		executor_contact TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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

	var exists bool
	var verified bool

	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		log.Fatal("Fail open Db при регистрации ", err)
		return err
	}

	if exists {
		err = DB.QueryRow("SELECT verified FROM users WHERE email = $1", email).Scan(&verified)
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

		_, err = DB.Exec(`UPDATE users SET password = $1, name = $2, verification_code = $3, time_live_code = $4
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
	_, err = DB.Exec(`INSERT INTO users(email, password, name, verified, verification_code, time_live_code) 
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
	_, err := DB.Exec(`UPDATE users SET verified = true, verification_code = NULL WHERE email = $1`, email)
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

	err := DB.QueryRow(`
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
	var storedHash string
	var verified bool
	var userID int

	err := DB.QueryRow(`
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

		_, err = DB.Exec(`
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
	Id          int    `json:"id"`
	Label       string `json:"label"`
	Discription string `json:"discription"`
	Avtor       string `json:"avtor"`
	Price       int    `json:"price"`
	Tag         string `json:"tag"`
	InFavorite  bool   `json:"infavorite"`
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

	var userId int
	err := DB.QueryRow(`
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

	rowsVakans, err := DB.Query(`
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

	rowsComments, err := DB.Query(`
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

func UpdateProf(name, password, tgUs string, recvizits int, email string) error {
	_, err := DB.Exec(`
		UPDATE users SET 
			name = $1,
			password = $2,
			tgUs = $3,
			recvizits = $4
		WHERE email = $5`,
		name, password, tgUs, recvizits, email)
	if err != nil {
		log.Println("Update error:", err)
		return err
	}
	return nil
}

func AddVakans(email, title, discription, tag string, price int) (error, string, int) {
	var userID int
	var avtor string
	err := DB.QueryRow("SELECT id, name FROM users WHERE email = $1", email).Scan(&userID, &avtor)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("user not found"), "", 0
		}
		return err, "", 0
	}

	_, err = DB.Exec(`
		INSERT INTO vakans (user_id, avtor, title, discription, price, tag)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, avtor, title, discription, price, tag)
	return err, avtor, userID
}

func GetVakans(email string, page int, tag, priceUpDownFalse string) ([]Vakans, error) {

	if page < 1 {
		page = 1
	}
	offset := (page - 1) * 60
	var user_id int

	if email != "" {
		proverka := DB.QueryRow("SELECT id FROM users WHERE email=$1", email).Scan(&user_id)
		if proverka != nil {
			log.Printf("❌ Ошибка EXISTS: %v", proverka)
			return nil, proverka
		}
	}
	var rows *sql.Rows
	var err error

	if tag != "" {
		switch priceUpDownFalse {
		case "Up":
			rows, err = DB.Query(`SELECT id, avtor, title, discription, price, tag FROM vakans WHERE tag = $1 ORDER BY price ASC LIMIT 60 OFFSET $2`, tag, offset)
		case "Down":
			rows, err = DB.Query(`SELECT id, avtor, title, discription, price, tag FROM vakans WHERE tag = $1 ORDER BY price DESC LIMIT 60 OFFSET $2`, tag, offset)
		default:
			rows, err = DB.Query(`SELECT id, avtor, title, discription, price, tag FROM vakans WHERE tag = $1 ORDER BY id ASC LIMIT 60 OFFSET $2`, tag, offset)
		}
	} else {
		switch priceUpDownFalse {
		case "Up":
			rows, err = DB.Query(`SELECT id, avtor, title, discription, price, tag FROM vakans ORDER BY price ASC LIMIT 60 OFFSET $1`, offset)
		case "Down":
			rows, err = DB.Query(`SELECT id, avtor, title, discription, price, tag FROM vakans ORDER BY price DESC LIMIT 60 OFFSET $1`, offset)
		default:
			rows, err = DB.Query(`SELECT id, avtor, title, discription, price, tag FROM vakans ORDER BY id ASC LIMIT 60 OFFSET $1`, offset)
		}
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vakansList []Vakans
	for rows.Next() {
		var v Vakans
		v.InFavorite = false
		err := rows.Scan(&v.Id, &v.Avtor, &v.Label, &v.Discription, &v.Price, &v.Tag)
		if err != nil {
			return nil, err
		}
		err = DB.QueryRow("SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND vakans_id = $2)", user_id, v.Id).Scan(&v.InFavorite)

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

func CreateOrder(vakansID int, clientEmail string) (int64, error) {
	var clientID int
	err := DB.QueryRow(`SELECT id FROM users WHERE email = $1`, clientEmail).Scan(&clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("user not found")
		}
		return 0, err
	}

	var vakans struct {
		Title        string
		Discription  string
		Price        int
		ExecutorID   int
		ExecutorName string
	}

	err = DB.QueryRow(`SELECT title, discription, price, user_id, avtor FROM vakans WHERE id = $1`, vakansID).Scan(
		&vakans.Title, &vakans.Discription, &vakans.Price, &vakans.ExecutorID, &vakans.ExecutorName)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("vakans not found")
		}
		return 0, err
	}

	if clientID == vakans.ExecutorID {
		return 0, errors.New("cannot buy your own vakans")
	}

	var balance int64
	err = DB.QueryRow("SELECT balance FROM users WHERE id = $1", clientID).Scan(&balance)
	if err != nil {
		return 0, err
	}
	if balance < int64(vakans.Price) {
		return 0, errors.New("insufficient funds")
	}

	tx, err := DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var orderID int64
	err = tx.QueryRow(`
		INSERT INTO orders (vakans_id, client_id, executor_id, title, description, price, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'frozen')
		RETURNING id`,
		vakansID, clientID, vakans.ExecutorID, vakans.Title, vakans.Discription, vakans.Price).Scan(&orderID)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`UPDATE users SET balance = balance - $1, 
		frozen_balance = frozen_balance + $1 WHERE id = $2`, vakans.Price, clientID)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`INSERT INTO transactions (user_id, order_id, type, amount, status)
		VALUES ($1, $2, 'freeze', $3, 'success')`, clientID, orderID, vakans.Price)
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return orderID, nil
}

func CompleteOrder(orderID int64) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var order struct {
		ExecutorID int
		Price      int
		ClientID   int
	}
	err = tx.QueryRow(`SELECT executor_id, price, client_id FROM orders WHERE id = $1 AND status = 'in_progress'`,
		orderID).Scan(&order.ExecutorID, &order.Price, &order.ClientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("order not found or not in progress")
		}
		return err
	}

	_, err = tx.Exec(`UPDATE users SET frozen_balance = frozen_balance - $1 WHERE id = $2`,
		order.Price, order.ClientID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE users SET balance = balance + $1, countSdelanihZakazov = countSdelanihZakazov + 1
		WHERE id = $2`, order.Price, order.ExecutorID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO transactions (user_id, order_id, type, amount, status)
		VALUES ($1, $2, 'payment', $3, 'success')`, order.ClientID, orderID, order.Price)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE orders SET status = 'completed', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, orderID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func CancelOrder(orderID int64) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var order struct {
		ClientID int
		Price    int
	}
	err = tx.QueryRow(`SELECT client_id, price FROM orders WHERE id = $1 AND status = 'frozen'`,
		orderID).Scan(&order.ClientID, &order.Price)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("order not found or cannot be cancelled")
		}
		return err
	}

	_, err = tx.Exec(`UPDATE users SET frozen_balance = frozen_balance - $1, balance = balance + $1 
		WHERE id = $2`, order.Price, order.ClientID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`INSERT INTO transactions (user_id, order_id, type, amount, status)
		VALUES ($1, $2, 'unfreeze', $3, 'success')`, order.ClientID, orderID, order.Price)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE orders SET status = 'cancelled', updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, orderID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func GetUserBalance(email string) (balance, frozen int64, err error) {
	err = DB.QueryRow(`SELECT balance, frozen_balance FROM users WHERE email = $1`, email).Scan(&balance, &frozen)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, errors.New("user not found")
		}
		return 0, 0, err
	}
	return balance, frozen, nil
}

func DepositSql(rubles int64, email string) error {
	_, err := DB.Exec("UPDATE users SET balance = balance + $1 WHERE email = $2", rubles, email)
	if err != nil {
		return err
	}
	log.Printf("User %s deposited %d kopecks", email, rubles)
	return nil
}

func GetFavorite(email string) ([]Vakans, error) {
	var user_id int
	err := DB.QueryRow("SELECT id FROM users WHERE email=$1", email).Scan(&user_id)
	if err != nil {
		return []Vakans{}, err
	}

	rows, err := DB.Query(`
		SELECT v.id, v.avtor, v.title, v.discription, v.price, v.tag 
		FROM favorites f 
		JOIN vakans v ON f.vakans_id = v.id 
		WHERE f.user_id = $1`, user_id)
	if err != nil {
		return []Vakans{}, err
	}
	defer rows.Close()

	var vakansList []Vakans
	for rows.Next() {
		var v Vakans
		err := rows.Scan(&v.Id, &v.Avtor, &v.Label, &v.Discription, &v.Price, &v.Tag)
		if err != nil {
			return []Vakans{}, err
		}
		vakansList = append(vakansList, v)
	}
	return vakansList, nil
}

func Like(email string, card_id int) error {
	var user_id int
	err := DB.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&user_id)
	if err != nil {
		return err
	}
	var card_in_favorite bool

	err = DB.QueryRow("SELECT EXISTS(SELECT 1 FROM favorites WHERE user_id = $1 AND vakans_id = $2)", user_id, card_id).Scan(&card_in_favorite)
	if err != nil {
		return err
	}

	if card_in_favorite == false {
		_, err := DB.Exec("INSERT INTO favorites (user_id, vakans_id) VALUES ($1, $2)", user_id, card_id)
		if err != nil {
			return errors.New("ошибка добавления в избранное")
		}
	} else if card_in_favorite == true {
		_, err := DB.Exec("DELETE FROM favorites WHERE user_id = $1 AND vakans_id = $2", user_id, card_id)
		if err != nil {
			return errors.New("ошибка удаления из избранного")
		}
	}
	return nil
}
