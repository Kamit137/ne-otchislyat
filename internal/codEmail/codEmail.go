package codemail

import (
	"fmt"
	"math/rand"

	"net/smtp"
	"time"
)

func SendVerificationCode(to, code string) error {
	from := "timaplay137@gmail.com"
	password := "kphvjxvbwzbjqhtx"
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	subject := "Код подтверждения регистрации"
	body := fmt.Sprintf(`
		<h2>Подтверждение email</h2>
		<p>Ваш код подтверждения: <strong>%s</strong></p>
		<p>Код действителен 15 минут.</p>
	`, code)

	msg := []byte("To: " + to + "\r\n" +
		"From: " + from + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" + body + "\r\n")

	auth := smtp.PlainAuth("", from, password, smtpHost)
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func GenerateCode() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := r.Intn(900000) + 100000
	return fmt.Sprintf("%d", code)
}
