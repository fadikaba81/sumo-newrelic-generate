package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

// ---------- Config ----------

var (
	port         = getenv("PORT", ":8443")
	certFile     = getenv("TLS_CERT_FILE", "./certs/fullchain.pem")
	keyFile      = getenv("TLS_KEY_FILE", "./certs/privkey.pem")
	sumoEndpoint = os.Getenv("SUMO_TOKEN")
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ---------- Log Structure ----------

type OrderLog struct {
	Timestamp     string `json:"timestamp"`
	Service       string `json:"service"`
	Env           string `json:"env"`
	Level         string `json:"level"`
	OrderID       string `json:"orderId"`
	HTTPCode      int    `json:"httpCode"`
	PortName      string `json:"portName"`
	Port          int    `json:"port"`
	CorrelationID string `json:"correlationId"`
	Message       string `json:"message"`
}

// ---------- Helpers ----------

func randomString(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func levelFromHTTPCode(code int) string {
	switch {
	case code >= 500:
		return "ERROR"
	case code >= 400:
		return "WARN"
	default:
		return "INFO"
	}
}

// ---------- Push To Sumo ----------

func pushToSumo(logEntry OrderLog) error {
	payload, err := json.Marshal(logEntry)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", sumoEndpoint, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Sumo-Category", "order-api/prod")
	req.Header.Set("X-Sumo-Name", "direct-push")
	req.Header.Set("X-Sumo-Host", "Order-API")

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("Sumo returned status: %s", resp.Status)
	}

	return nil
}

// Generate a random Correlation-ID
func generateCorrelationID(n int) string {
	genID := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)

	for i := range b {
		b[i] = genID[rand.Intn(len(genID))]
	}

	return string(b)
}

// Get a random environment name
func randEnv() string {
	envs := []string{"Prod", "PTEST", "STEST", "VTEST"}
	return envs[rand.Intn(len(envs))]
}

func randHTTPCode() int {
	r := rand.Float64()
	switch {
	case r < 0.60:
		success := []int{200, 201}
		return success[rand.Intn(len(success))]
	case r < 0.80:
		clientErr := []int{400, 401, 403}
		return clientErr[rand.Intn(len(clientErr))]
	default:
		serverErr := []int{500, 502, 503, 504}
		return serverErr[rand.Intn(len(serverErr))]
	}
}

// Return the HTTP code, with extra server errors during the first 3 minutes of every 5
func generateHTTPCode() int {
	now := time.Now().UTC()

	if now.Minute()%5 <= 2 {
		if rand.Float64() < 0.80 {
			errors := []int{500, 502, 503, 504}
			return errors[rand.Intn(len(errors))]
		}
	}

	return randHTTPCode()
}

// ---------- HTTP Handler ----------

func orderHandler(w http.ResponseWriter, r *http.Request) {
	correlationID := r.Header.Get("X-Correlation-ID")

	if correlationID == "" {
		correlationID = generateCorrelationID(12)
	}

	// Simulate occasional error
	code := generateHTTPCode()

	entry := OrderLog{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Service:       "order-api",
		Env:           randEnv(),
		Level:         levelFromHTTPCode(code),
		OrderID:       randomString(8),
		CorrelationID: correlationID,
		HTTPCode:      code,
		PortName:      "HTTPS",
		Port:          443,
		Message:       "order processed",
	}
	log.Printf("CorrelationID being sent: %s", entry.CorrelationID)

	// Push to Sumo
	if err := pushToSumo(entry); err != nil {
		log.Printf("Push failed: %v", err)
	}

	w.Header().Set("X-Correlation-ID", correlationID)
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(entry)
}

// ---------- Main ----------

func main() {
	rand.Seed(time.Now().UnixNano())

	if sumoEndpoint == "" {
		log.Fatal("SUMO_TOKEN environment variable not set")
	}

	http.HandleFunc("/", orderHandler)

	server := &http.Server{
		Addr: port,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	log.Printf("Starting HTTPS server on %s (cert: %s)", port, certFile)
	log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
}