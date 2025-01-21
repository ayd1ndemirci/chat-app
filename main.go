package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type Message struct {
    ID        int    `json:"id"`
    Username  string `json:"username"`
    Message   string `json:"message"`
    CreatedAt string `json:"created_at"`
}


func main() {
	defer recoverPanic()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Environment dosyası yüklenemedi:", err)
	}

	db, err := getDBConnection()
	if err != nil {
		log.Fatal("MySQL bağlantı hatası:", err)
	}
	defer db.Close()

	fmt.Println("MySQL connected")

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/chat", chatHandler)
	http.HandleFunc("/messages", messagesHandler)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	fmt.Println("Sunucu çalışıyor: http://localhost:8080")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getDBConnection() (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_NAME"),
	)
	return sql.Open("mysql", dsn)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	
	http.Redirect(w, r, "/chat", http.StatusFound)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		if username == "" {
			http.Error(w, "Kullanıcı adı boş olamaz", http.StatusBadRequest)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:  "username",
			Value: username,
			Path:  "/",
		})

		http.Redirect(w, r, "/chat", http.StatusFound)
		return
	}

	tmpl, _ := template.ParseFiles("templates/login.html")
	tmpl.Execute(w, nil)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	username, err := r.Cookie("username")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	tmpl, _ := template.ParseFiles("templates/chat.html")
	tmpl.Execute(w, map[string]string{"Username": username.Value})
}

func messagesHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := getDBConnection()
	if err != nil {
		http.Error(w, "Veritabanı bağlantısı başarısız", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	if r.Method == http.MethodPost {
		username, _ := r.Cookie("username")
		message := r.FormValue("message")

		_, err := conn.Exec("INSERT INTO messages (username, message) VALUES (?, ?)", username.Value, message)
		if err != nil {
			http.Error(w, "Mesaj kaydedilemedi", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	rows, err := conn.Query("SELECT id, username, message, created_at FROM messages ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, "Mesajlar alınamadı", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.Username, &msg.Message, &msg.CreatedAt); err != nil {
			http.Error(w, "Mesajları işlerken hata oluştu", http.StatusInternalServerError)
			return
		}
		messages = append(messages, msg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func recoverPanic() {
	if r := recover(); r != nil {
		log.Printf("Program çöktü: %v", r)
	}
}
