package main

import (
	"bufio"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

//go:embed index.html
var indexHTML embed.FS

type webhookConfig struct {
	ngrokStaticDomain string `env:"NGROK_STATIC_DOMAIN"`
}

// LoggingResponseWriter는 http.ResponseWriter를 래핑하여 상태 코드와 응답 길이를 캡처합니다.
type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	length     int
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// 요청 정보 로깅
		log.Printf("Started %s %s", r.Method, r.URL.Path)

		// Response writer를 래핑하여 상태 코드와 응답 길이를 캡처
		lrw := NewLoggingResponseWriter(w)
		next.ServeHTTP(lrw, r)

		// 요청 처리 완료 후 로깅
		duration := time.Since(start)
		log.Printf("Completed in %v %s %s %d %d", duration, r.Method, r.URL.Path, lrw.statusCode, lrw.length)
	})
}

func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{ResponseWriter: w}
}

func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	if lrw.statusCode == 0 {
		lrw.statusCode = code
	}
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *LoggingResponseWriter) Write(b []byte) (int, error) {
	// 상태 코드가 설정되어 있지 않다면, 기본값인 200으로 설정합니다.
	if lrw.statusCode == 0 {
		lrw.statusCode = http.StatusOK
		lrw.ResponseWriter.WriteHeader(lrw.statusCode)
	}
	n, err := lrw.ResponseWriter.Write(b)
	lrw.length += n
	return n, err
}

var ngrokCmd *exec.Cmd

func startNgrok(cfg *webhookConfig) error {
	cmd := exec.Command("ngrok", "http", "--domain="+cfg.ngrokStaticDomain, "8081", "--log=stdout")
	ngrokCmd = cmd

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	re := regexp.MustCompile(`https://[\w\-]+\.ngrok-free\.app`)

	for scanner.Scan() {
		line := scanner.Text()
		log.Println("ngrok stdout:", line)
		if matches := re.FindStringSubmatch(line); matches != nil {
			url := matches[0]
			log.Println("ngrok URL:", url)

			cmd := exec.Command("sh", "-c", fmt.Sprintf("printf %s | pbcopy", strconv.Quote(url)))
			if err := cmd.Run(); err != nil {
				log.Fatalf("Failed to copy ngrok URL to clipboard: %s", err)
			} else {
				log.Println("ngrok URL copied to clipboard.")
			}
			return nil
		}
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return fmt.Errorf("ngrok URL not found")
}

// stopNgrok 함수는 NGROK 세션을 종료합니다.
func stopNgrok() {
	if ngrokCmd != nil {
		log.Println("Stopping ngrok...")
		if err := ngrokCmd.Process.Kill(); err != nil {
			log.Fatalf("Failed to stop ngrok: %s", err)
		}
	}
}

// rootHandler 함수는 루트 경로 요청을 처리합니다.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// index.html 파일의 내용을 읽어 클라이언트에게 전송
		htmlContent, err := indexHTML.ReadFile("index.html")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Println("Error reading index.html:", err)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(htmlContent)
	} else {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
	}
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		log.Printf("Webhook received: %+v", payload)

		jsonData, err := json.Marshal(payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fileName := fmt.Sprintf("webhook_logs_%s.json", time.Now().Format("2006-01-02"))
		file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Printf("Failed to open file: %s", err)
			http.Error(w, "Failed to write to file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		if _, err := file.Write(jsonData); err != nil {
			log.Printf("Failed to write to file: %s", err)
			http.Error(w, "Failed to write to file", http.StatusInternalServerError)
			return
		}
		if _, err := file.WriteString("\n"); err != nil {
			log.Printf("Failed to write newline to file: %s", err)
			http.Error(w, "Failed to write to file", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Webhook received successfully"))
	} else {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
	}
}

func webhook5xxHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Fake Internal Server Error", http.StatusInternalServerError)
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	fileName := fmt.Sprintf("webhook_logs_%s.json", date)
	file, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "No logs found for the specified date", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	var logs []map[string]interface{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var logEntry map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &logEntry); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logs = append(logs, logEntry)
	}

	if err := scanner.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		log.Printf("Error encoding logs to JSON: %v", err)
		http.Error(w, "Failed to encode logs", http.StatusInternalServerError)
	}
}

func readExistingLogs() {
	date := time.Now().Format("2006-01-02")
	fileName := fmt.Sprintf("webhook_logs_%s.json", date)
	log.Printf("Attempting to open file: %s", fileName)
	file, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("No existing log file found: %s", fileName)
			_, err := os.Create(fileName)
			if err != nil {
				log.Fatalf("Failed to create new log file: %s", err)
			}
			log.Printf("Created new log file: %s", fileName)
			return
		}
		log.Fatalf("Failed to open existing log file: %s", err)
	}
	defer file.Close()
	log.Println("File opened successfully")

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		message := scanner.Text()
		if message == "" {
			continue // 빈 줄 건너뛰기
		}
		log.Printf("Read message: %s", message)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Failed to read existing log file: %s", err)
	}
	log.Println("Finished reading existing log file")
}

func clearLogsHandler(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		http.Error(w, "Date parameter is required", http.StatusBadRequest)
		return
	}
	fileName := fmt.Sprintf("webhook_logs_%s.json", date)
	err := os.Remove(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "No logs found for the specified date", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Logs cleared successfully"))
}

func handleRequests() {
	http.Handle("/", loggingMiddleware(http.HandlerFunc(rootHandler)))
	http.Handle("/webhook", loggingMiddleware(http.HandlerFunc(webhookHandler)))
	http.Handle("/webhook500", loggingMiddleware(http.HandlerFunc(webhook5xxHandler)))
	http.Handle("/logs", loggingMiddleware(http.HandlerFunc(logsHandler)))
	http.Handle("/clear_logs", loggingMiddleware(http.HandlerFunc(clearLogsHandler)))
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	cfg := &webhookConfig{
		ngrokStaticDomain: os.Getenv("NGROK_STATIC_DOMAIN"),
	}

	// ngrok 무료 account는 한개의 세션만 허용하므로, 이전 세션을 종료합니다.
	stopNgrok()

	log.Println("Attempting to start ngrok...")
	err = startNgrok(cfg)
	if err != nil {
		log.Fatalf("Failed to start ngrok: %s", err)
	}

	handleRequests()

	server := &http.Server{
		Addr: ":8081",
	}

	go func() {
		log.Println("Starting server on :8081...")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %s\n", err)
		}
	}()

	readExistingLogs()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %s", err)
	}

	stopNgrok()

	log.Println("Server exiting")
}
