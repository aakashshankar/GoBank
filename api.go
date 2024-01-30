package main

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type apiFunc func(*http.ResponseWriter, *http.Request) error

type APIError struct {
	Error string `json:"error"`
}

type APIServer struct {
	listenAddr string
	store      Persistence
}

func NewAPIServer(listenAddr string, store Persistence) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store:      store,
	}
}

func (s *APIServer) Run() {
	log.Println("Staring API server on", s.listenAddr)
	router := mux.NewRouter()

	router.HandleFunc("/create", httpHandler(s.post)).Methods(http.MethodPost)
	router.HandleFunc("/login", httpHandler(s.login)).Methods(http.MethodPost)
	router.HandleFunc("/accounts/{id}", WithJWTAuth(httpHandler(s.get), s.store)).Methods(http.MethodGet)
	router.HandleFunc("/list_accounts", httpHandler(s.list)).Methods(http.MethodGet)
	router.HandleFunc("/accounts/{id}/delete", WithJWTAuth(httpHandler(s.delete), s.store)).Methods(http.MethodDelete)
	router.HandleFunc("/transfer", WithJWTAuth(httpHandler(s.transfer), s.store)).Methods(http.MethodPost)

	err := http.ListenAndServe(s.listenAddr, router)

	if err != nil {
		log.Fatal("Error when starting API server: ", err)
		return
	}
}

func (s *APIServer) login(w *http.ResponseWriter, r *http.Request) error {
	loginRequest := &LoginRequest{}
	if err := json.NewDecoder(r.Body).Decode(loginRequest); err != nil {
		return err
	}
	account, err := s.store.GetByNumber(loginRequest.Number)
	if err != nil {
		return err
	}

	if !CompareHashedPassword(loginRequest.Password, account.Password) {
		return WriteJSON(*w, http.StatusUnauthorized, APIError{Error: "Incorrect password!"})
	}

	tokenString, _ := generateJWT(account)
	(*w).Header().Set("Authorization", "Bearer "+tokenString)

	return WriteJSON(*w, http.StatusOK, loginRequest)
}

func (s *APIServer) list(w *http.ResponseWriter, r *http.Request) error {
	accounts, err := s.store.List()
	if err != nil {
		return err
	}
	return WriteJSON(*w, http.StatusOK, accounts)
}

func (s *APIServer) get(w *http.ResponseWriter, r *http.Request) error {
	pathParams := mux.Vars(r)
	id, _ := strconv.Atoi(pathParams["id"])
	account, err := s.store.Get(id)
	if err != nil {
		return err
	}
	return WriteJSON(*w, http.StatusOK, account)
}

func (s *APIServer) post(w *http.ResponseWriter, r *http.Request) error {
	createAccReq := &CreateAccountRequest{}
	if err := json.NewDecoder(r.Body).Decode(createAccReq); err != nil {
		return err
	}
	hashedPasswd, _ := HashPassword(createAccReq.Password)
	account := NewAccount(createAccReq.FirstName, createAccReq.LastName, hashedPasswd)
	if err := s.store.Save(account); err != nil {
		return err
	}

	tokenString, err := generateJWT(account)

	if err != nil {
		return err
	}

	(*w).Header().Set("Authorization", "Bearer "+tokenString)

	fmt.Println("JWT Token: ", tokenString)

	return WriteJSON(*w, http.StatusOK, account)
}

func (s *APIServer) delete(w *http.ResponseWriter, r *http.Request) error {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if err := s.store.Delete(id); err != nil {
		return err
	}
	return WriteJSON(*w, http.StatusOK, map[string]int{"deleted": id})
}

func (s *APIServer) transfer(w *http.ResponseWriter, r *http.Request) error {
	transferReq := &TransferRequest{}
	if err := json.NewDecoder(r.Body).Decode(transferReq); err != nil {
		return err
	}
	// waits for the transfer method to finish then executes
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error when closing body: ", err)
		}
	}(r.Body)

	return WriteJSON(*w, http.StatusOK, transferReq)
}
func PermissionDenied(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	err := WriteJSON(w, http.StatusUnauthorized, APIError{Error: "Permission denied"})
	if err != nil {
		log.Printf("Error when writing JSON: %v", err)
	}
}

func WithJWTAuth(handlerFunc http.HandlerFunc, s Persistence) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Authenticating user")
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) == 0 {
			PermissionDenied(w)
			return
		}

		token, ok := validateJWT(r.Header.Get("Authorization"))

		if ok != nil {
			log.Println("Error validating JWT: ", ok)
			PermissionDenied(w)
			return
		}

		if !token.Valid {
			PermissionDenied(w)
			return
		}

		userId, _ := strconv.Atoi(mux.Vars(r)["id"])
		acc, err := s.Get(userId)
		log.Println("Account: ", acc)

		claims := token.Claims.(jwt.MapClaims)
		if err != nil {
			log.Println("Error getting account: ", err)
			PermissionDenied(w)
			return
		}
		if acc.Number != int64(claims["account"].(float64)) {
			log.Println("Cross account access detected")
			PermissionDenied(w)
			return
		}

		handlerFunc(w, r)
	}
}

func generateJWT(account *Account) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))

	claims := &jwt.MapClaims{
		"expiresAt": time.Now().Add(time.Minute * 15).Unix(),
		"account":   account.Number,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func validateJWT(tokenString string) (*jwt.Token, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CompareHashedPassword(passwd string, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(passwd))
	return err == nil
}

func WriteJSON(w http.ResponseWriter, status int, v interface{}) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func httpHandler(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(&w, r); err != nil {
			err := WriteJSON(w, http.StatusBadRequest, APIError{Error: err.Error()})
			if err != nil {
				log.Printf("Error when writing JSON: %v", err)
				return
			}
		}
	}
}
