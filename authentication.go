package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

type User struct {
	Id       uint64 `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Token struct {
	AccessToken  string
	RefreshToken string
	AccessUuid   string
	RefreshUuid  string
	AtExpires    int64
	RtExpires    int64
}

type AccessDetails struct {
	AccessUuid string
	UserId     uint64
}

type authentication struct {
	AccessExpireDuration  time.Duration
	RefreshExpireDuration time.Duration
	AccessSecret          string
	RefreshSecret         string
	db                    *sql.DB
	userStmt              *sql.Stmt
	expirationCtx         context.Context
}

func (amw *authentication) Initialize() {
	amw.AccessExpireDuration = time.Minute * 15
	amw.RefreshExpireDuration = time.Hour * 24 * 7

	amw.AccessSecret = os.Getenv("ACCESS_SECRET")
	amw.RefreshSecret = os.Getenv("REFRESH_SECRET")

	config := mysql.NewConfig()
	config.User = os.Getenv("MYSQL_USER")
	config.Passwd = os.Getenv("MYSQL_PASS")
	config.Addr = "localhost:3306"
	config.DBName = "guptaspi"

	var err error
	amw.db, err = sql.Open("mysql", config.FormatDSN())
	if err != nil {
		panic(err)
	}

	amw.db.SetConnMaxLifetime(time.Minute * 3)
	amw.db.SetMaxOpenConns(10)
	amw.db.SetMaxIdleConns(10)

	amw.userStmt, err = amw.db.Prepare("SELECT id, username, password FROM users WHERE username = ?")
	if err != nil {
		log.Fatalf("Error creating prepared statement: %v\n", err)
	}

	log.Printf("Connected to MYSQL db")

	amw.expirationCtx = context.TODO()
	go amw.deleteExpired(amw.expirationCtx)
}

func (amw *authentication) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == "/login" || r.RequestURI == "/createUser" {
			next.ServeHTTP(w, r)
			return
		}
		err := tokenValid(r)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractTokenMetadata(r *http.Request) (*AccessDetails, error) {
	token, err := verifyToken(r)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		accessUuid, ok := claims["access_uuid"].(string)
		if !ok {
			return nil, err
		}
		userId, err := strconv.ParseUint(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)
		if err != nil {
			return nil, err
		}
		return &AccessDetails{
			AccessUuid: accessUuid,
			UserId:     userId,
		}, nil
	}
	return nil, err
}

func tokenValid(r *http.Request) error {
	token, err := verifyToken(r)
	if err != nil {
		return err
	}
	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		return err
	}
	return nil
}

func verifyToken(r *http.Request) (*jwt.Token, error) {
	tokenString := extractToken(r)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("ACCESS_SECRET")), nil
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}

func extractToken(r *http.Request) string {
	bearToken := r.Header.Get("Authorization")
	strArr := strings.Split(bearToken, " ")
	if len(strArr) == 2 {
		return strArr[1]
	}
	return ""
}

func (amw *authentication) CreateUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	body := struct {
		UserName string `json:"user_name"`
		Password string `json:"password"`
	}{}
	err := decoder.Decode(&body)
	if err != nil {
		log.Printf("Error decoding JSON: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = amw.db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", body.UserName, hash)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (amw *authentication) Login(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	if !ok {
		log.Printf("Bad basic auth header")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	user := User{}

	err := amw.userStmt.QueryRow(username).Scan(&user.Id, &user.Username, &user.Password)
	switch {
	case err == sql.ErrNoRows:
		w.WriteHeader(http.StatusUnauthorized)
		return
	case err != nil:
		log.Printf("Error when querying users: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	} else {
		token, err := amw.createToken(user.Id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = amw.createAuth(user.Id, token)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		tokens := map[string]string{
			"access_token":  token.AccessToken,
			"refresh_token": token.RefreshToken,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokens)
	}
}

func (amw *authentication) Logout(w http.ResponseWriter, r *http.Request) {
	au, err := extractTokenMetadata(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	err = amw.deleteAuth(au.AccessUuid)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (amw *authentication) deleteAuth(givenUuid string) error {
	_, err := amw.db.Exec("DELETE FROM access_tokens WHERE access_uuid = ?", givenUuid)
	if err != nil {
		return err
	}
	return nil
}

func (amw *authentication) createToken(userId uint64) (*Token, error) {
	td := &Token{
		AtExpires: time.Now().Add(amw.AccessExpireDuration).Unix(),
		RtExpires: time.Now().Add(amw.RefreshExpireDuration).Unix(),
	}

	if id, err := uuid.NewRandom(); err != nil {
		return nil, err
	} else {
		td.AccessUuid = id.String()
	}
	if id, err := uuid.NewRandom(); err != nil {
		return nil, err
	} else {
		td.RefreshUuid = id.String()
	}

	var err error
	// Create access token
	atClaims := jwt.MapClaims{}
	atClaims["authorized"] = true
	atClaims["access_uuid"] = td.AccessUuid
	atClaims["user_id"] = userId
	atClaims["exp"] = td.AtExpires
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	td.AccessToken, err = at.SignedString([]byte(amw.AccessSecret))
	if err != nil {
		return nil, err
	}

	// Create refresh token
	rtClaims := jwt.MapClaims{}
	rtClaims["refresh_uuid"] = td.RefreshUuid
	rtClaims["user_id"] = userId
	rtClaims["exp"] = td.RtExpires
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = rt.SignedString([]byte(amw.RefreshSecret))
	if err != nil {
		return nil, err
	}

	return td, nil
}

func (amw *authentication) createAuth(userid uint64, td *Token) error {
	at := time.Unix(td.AtExpires, 0)
	rt := time.Unix(td.RtExpires, 0)

	tx, err := amw.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO access_tokens (user_id, access_uuid, expires) VALUES (?, ?, ?)", userid, td.AccessUuid, at)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec("INSERT INTO refresh_tokens (user_id, refresh_uuid, expires) VALUES (?, ?, ?)", userid, td.RefreshUuid, rt)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return nil
}

func (amw *authentication) Refresh(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	body := struct {
		RefreshToken string `json:"refresh_token"`
	}{}
	err := decoder.Decode(&body)
	if err != nil {
		log.Printf("Error decoding JSON: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	refreshToken := body.RefreshToken

	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(amw.RefreshSecret), nil
	})

	if err != nil {
		log.Printf("Error getting token: %v\n", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if _, ok := token.Claims.(jwt.Claims); !ok && !token.Valid {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Token is valid, get the uuid
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		refreshUuid, ok := claims["refresh_uuid"].(string)
		if !ok {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		userId, err := strconv.ParseUint(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)
		if err != nil {
			log.Printf("Error getting user ID: %v\n", err)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		_, err = amw.db.Exec("DELETE FROM refresh_tokens WHERE refresh_uuid = ?", refreshUuid)
		if err != nil {
			log.Printf("Error deleting previous Refresh Token: %v\n", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ts, err := amw.createToken(userId)
		if err != nil {
			log.Printf("Error creating new token pairs: %v\n", err)
			w.WriteHeader(http.StatusForbidden)
			return
		}

		err = amw.createAuth(userId, ts)
		if err != nil {
			log.Printf("Error saving token pairs: %v\n", err)
			w.WriteHeader(http.StatusForbidden)
			return
		}

		tokens := map[string]string{
			"access_token":  ts.AccessToken,
			"refresh_token": ts.RefreshToken,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(tokens)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
}

func (amw *authentication) deleteExpired(ctx context.Context) {
	log.Printf("Starting deletion of expired tokens...")
	stmt, err := amw.db.Prepare("DELETE FROM access_tokens WHERE expires < ?")
	stmt2, err2 := amw.db.Prepare("DELETE FROM refresh_tokens WHERE expires < ?")
	if err != nil {
		log.Fatalf("Error creating prepared statement: %v\n", err)
	}
	if err2 != nil {
		log.Fatalf("Error creating prepared statement: %v\n", err2)
	}
	for {
		select {
		case <-ctx.Done():
			log.Printf("Stopping deletion of expired tokens...")
		default:
			_, err = stmt.Exec(time.Now())
			if err != nil {
				log.Printf("Error deleting rows: %v\n", err)
			}
			_, err = stmt2.Exec(time.Now())
			if err != nil {
				log.Printf("Error deleting rows: %v\n", err)
			}
			time.Sleep(5 * time.Minute)
		}
	}
}
