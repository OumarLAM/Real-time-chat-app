package main

import (
	"database/sql"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

var jwtSecret = []byte("secretKey")
var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Message struct {
	// Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

var db *sql.DB

func main() {
	//Open SQL database connection
	db, _ = sql.Open("sqlite3", "./db.sqlite")
	defer db.Close()

	// Initialize database tables
	createTable()

	//Handle routes
	http.HandleFunc("/", authMiddleware(indexHandler))
	http.HandleFunc("/home", homeHandler)
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/register", registerHandler)

	go handleMessages()

	fmt.Println("Server is running on http://localhost:8000")
	http.ListenAndServe(":8000", nil)
}

// Create JWT token for a user
func createToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp": time.Now().Add(time.Hour * 24).Unix(),	// Token expires in one day
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err!= nil {
        return "", err
    }

	return tokenString, nil
}

func createTable() {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT,
        password TEXT
	);
	`
	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Println("Error creating table: ", err)
	}
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenCookie, err := r.Cookie("token")
		if err != nil || tokenCookie == nil {
			//No token found, redirect to login
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		tokenString := tokenCookie.Value
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Token is valid, call the next handler
		next.ServeHTTP(w, r)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "index", nil)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Query the database to get the hashed password for the given username
		row := db.QueryRow("SELECT password FROM users WHERE username = ?", username)

		var hashedPassword string
		err := row.Scan(&hashedPassword)
		if err != nil {
			// User not found or an error occurred
			log.Printf("Login failed for user %s: %v", username, err)
			renderTemplate(w, "login", map[string]interface{}{"Error": "Invalid credentials"})
			return
		}

		// Compare the hashed password from the database with the provided password
		err2 := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
		if err2 != nil {
			// Passwords don't match
			log.Printf("Login failed for user %s: %v", username, err2)
			renderTemplate(w, "login", map[string]interface{}{"Error": "Invalid credentials"})
			return
		}

		// Passwords match, create a token and send it to the client
		log.Println(username, " is successfully logged in")
		token, err := createToken(username)
		if err != nil {
			log.Printf("Error creating token for user %s: %v", username, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Set the token as a cookie or send it in the response as needed
		http.SetCookie(w, &http.Cookie{
			Name: "token",
			Value: token,
			Expires: time.Now().Add(time.Hour * 24),
			Path: "/",
		})

		http.Redirect(w, r, "/home", http.StatusSeeOther)
	} else {
		renderTemplate(w, "login", nil)
	}
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Hash the password before storing it in the database
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Error hashing password for user %s: %v", username, err)
			renderTemplate(w, "register", map[string]interface{}{"Error": "Error registering user"})
			return
		}

		// Insert the hashed password into the database
		_, err = db.Exec("INSERT INTO users (username, password) VALUES (?,?)", username, string(hashedPassword))
		if err != nil {
			log.Printf("Error registering user %s: %v", username, err)
			renderTemplate(w, "register", map[string]interface{}{"Error": "Error registering user"})
			return
		}

		// Registration successful, create a token and send it to the client
		token, err := createToken(username)
        if err!= nil {
            log.Printf("Error creating token for user %s: %v", username, err)
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        // Set the token as a cookie or send it in the response as needed
        http.SetCookie(w, &http.Cookie{
            Name: "token",
            Value: token,
            Expires: time.Now().Add(time.Hour * 24),
            Path: "/",
        })

		http.Redirect(w, r, "/home", http.StatusSeeOther)
	} else {
		renderTemplate(w, "register", nil)
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, err := template.ParseFiles("public/" + tmpl + ".html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer ws.Close()

	clients[ws] = true

	for {
		var msg Message

		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error: %v", err)
			delete(clients, ws)
			break
		}

		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast

		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("Error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
