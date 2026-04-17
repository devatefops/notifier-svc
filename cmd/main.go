package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the application.
type Config struct {
	CounterSvcHost string
	SMTPServer     string
	SMTPPort       int
	SMTPUser       string
	SMTPPass       string
	EmailTo        string
	CheckInterval  time.Duration
}

// CounterResponse matches the counter_service API response
type CounterResponse struct {
	Value int `json:"value"`
}

func main() {
	// Validate and load SMTP port
	smtpPortStr := os.Getenv("SMTP_PORT")
	if smtpPortStr == "" {
		log.Fatal("SMTP_PORT is not set")
	}
	smtpPort, err := strconv.Atoi(smtpPortStr)
	if err != nil {
		log.Fatalf("invalid SMTP port: %v", err)
	}

	// Validate and load check interval
	intervalStr := os.Getenv("CHECK_INTERVAL")
	if intervalStr == "" {
		log.Fatal("CHECK_INTERVAL is not set")
	}
	checkInterval, err := time.ParseDuration(intervalStr)
	if err != nil {
		log.Fatalf("invalid CHECK_INTERVAL: %v", err)
	}

	cfg := Config{
		CounterSvcHost: os.Getenv("COUNTER_SVC_HOST"),
		SMTPServer:     os.Getenv("SMTP_HOST"),
		SMTPPort:       smtpPort,
		SMTPUser:       os.Getenv("SMTP_USER"),
		SMTPPass:       os.Getenv("SMTP_PASS"),
		EmailTo:        os.Getenv("EMAIL_TO"),
		CheckInterval:  checkInterval,
	}

	// Basic config validation
	if cfg.CounterSvcHost == "" || cfg.SMTPServer == "" || cfg.SMTPUser == "" ||
		cfg.SMTPPass == "" || cfg.EmailTo == "" {
		log.Fatal("One or more required environment variables are missing")
	}

	// Send a welcome email on startup
	sendWelcomeEmail(cfg)

	// Periodic check loop
	for {
		checkCounterAndNotify(cfg)
		log.Printf("Check complete. Sleeping for %s...", cfg.CheckInterval)
		time.Sleep(cfg.CheckInterval)
	}
}

// checkCounterAndNotify fetches the counter value and sends an email if it reaches 10
func checkCounterAndNotify(cfg Config) {
	url := fmt.Sprintf("http://%s/api/counter", cfg.CounterSvcHost)

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Failed to call counter service: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Counter service returned non-OK status: %s", resp.Status)
		return
	}

	var counterResp CounterResponse
	if err := json.NewDecoder(resp.Body).Decode(&counterResp); err != nil {
		log.Printf("Failed to decode counter response: %v", err)
		return
	}

	count := counterResp.Value
	log.Printf("Current counter value is %d", count)

	if count != 10 {
		log.Printf("Counter is not 10 (value: %d), skipping email.", count)
		return
	}

	subject := "Counter Alert!"
	body := fmt.Sprintf("The counter has reached the target value of %d.", count)

	if err := sendEmail(cfg, subject, body); err != nil {
		log.Printf("Failed to send notification email: %v", err)
	} else {
		log.Println("Notification email sent successfully!")
	}
}

// sendWelcomeEmail sends an email when the service starts
func sendWelcomeEmail(cfg Config) {
	log.Println("Sending welcome email...")
	subject := "Notifier Service Started"
	body := "Welcome! The notifier service is running and will alert you when the counter reaches 10."

	if err := sendEmail(cfg, subject, body); err != nil {
		log.Printf("Failed to send welcome email: %v", err)
	} else {
		log.Println("Welcome email sent successfully!")
	}
}

// sendEmail sends an email using SMTP
func sendEmail(cfg Config, subject, body string) error {
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPServer)

	to := []string{cfg.EmailTo}
	msg := []byte(
		"To: " + cfg.EmailTo + "\r\n" +
			"From: " + cfg.SMTPUser + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			"<html><body><p>" + body + "</p></body></html>\r\n",
	)

	addr := fmt.Sprintf("%s:%d", cfg.SMTPServer, cfg.SMTPPort)
	return smtp.SendMail(addr, auth, cfg.SMTPUser, to, msg)
}
