package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth"
	_ "github.com/mattn/go-sqlite3"
	"net/http"
	"strings"
	"time"
)

var tokenAuth *jwtauth.JWTAuth

func init() {
	tokenAuth = jwtauth.New("HS256", []byte("secret"), nil)
}

func main() {
	addr := ":3001"
	fmt.Printf("Server started at %v\n", addr)
	http.ListenAndServe(addr, router())
}

func router() http.Handler {
	r := chi.NewRouter()

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator)

		r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
			_, claims, _ := jwtauth.FromContext(r.Context())

			var exp int64
			if expv, ok := claims["exp"]; ok {
				switch v := expv.(type) {
				case float64:
					exp = int64(v)
				case int64:
					exp = v
				case json.Number:
					exp, _ = v.Int64()
				default:
				}
			}
			w.Write([]byte(fmt.Sprintf("Protected area. Welcome: %v\nYour Token expires at: %v\n", claims["user_id"], time.Unix(exp, 0))))
		})
	})

	// Public routes
	r.Group(func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Welcome anonymous\n"))
		})
		r.Get("/auth", func(w http.ResponseWriter, r *http.Request) {
			auth := strings.SplitN(r.Header.Get("Authorization"), " ", 2)

			if len(auth) != 2 || auth[0] != "Basic" {
				http.Error(w, "Authorization failed\n", http.StatusUnauthorized)
				return
			}

			payload, _ := base64.StdEncoding.DecodeString(auth[1])
			pair := strings.SplitN(string(payload), ":", 2)

			if len(pair) != 2 || !validate(pair[0], pair[1]) {
				http.Error(w, "Authorization falied: invalid credentials\n", http.StatusUnauthorized)
			} else {
				//The JWT will have 180secs of lifetime
				expiration := int64(time.Now().Unix()) + 180
				w.Write([]byte(fmt.Sprintf("This is your JWT for this session: %v\nExpires at: %s\n\n", generateJwt(pair[0], expiration), time.Unix(expiration, 0))))
			}
		})
	})

	return r
}

//Users validation
func validate(username, password string) bool {
	var out bool
	database, _ := sql.Open("sqlite3", "./users.db")
	err := database.QueryRow("select username, password from users where username LIKE ? and password LIKE ?", username, password).Scan(&username)
	if err != nil && err == sql.ErrNoRows {
		out = false
	} else {
		out = true
	}
	return out
}

//JWT generation
func generateJwt(username string, expiration int64) string {
	_, tokenString, _ := tokenAuth.Encode(jwtauth.Claims{"user_id": username, "exp": expiration})
	fmt.Printf("DEBUG: JWT: %s\n Claim: \"user_id\": %s\n\"exp\": %s", tokenString, username, time.Unix(expiration, 0))
	return tokenString
}
