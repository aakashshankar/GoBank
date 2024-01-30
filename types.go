package main

import (
	"math/rand"
	"time"
)

type CreateAccountRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Password  string `json:"password"`
}

type LoginRequest struct {
	Number   int64  `json:"accountNumber"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Number int64 `json:"accountNumber"`
}

type Account struct {
	ID        int       `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Number    int64     `json:"accountNumber"`
	Password  string    `json:"-"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"createdAt"`
}

type TransferRequest struct {
	To     int64 `json:"toAccount"`
	Amount int64 `json:"amount"`
}

func NewAccount(firstName string, lastName string, hashedPasswd string) *Account {
	return &Account{
		FirstName: firstName,
		LastName:  lastName,
		Number:    rand.Int63n(1000000000000000),
		Password:  hashedPasswd,
		CreatedAt: time.Now().UTC(),
	}
}
