package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

type Persistence interface {
	Save(*Account) error

	Delete(int) error

	Update(*Account) error

	Get(int) (*Account, error)

	List() ([]*Account, error)

	GetByNumber(number int64) (*Account, error)
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	connStr := "user=postgres dbname=gobank sslmode=disable"
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStore{
		db: db,
	}, nil
}

func (s *PostgresStore) init() error {
	if err := s.createAccountsTable(); err != nil {
		return err
	}

	return nil
}

func (s *PostgresStore) createAccountsTable() error {
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS accounts (
		id serial primary key, 
		first_name varchar(255), 
		last_name varchar(255), 
		number bigint,
		password text,
		balance bigint,
		created_at timestamp default current_timestamp
		)`)

	return err
}

func (s *PostgresStore) Save(a *Account) error {
	resp, err := s.db.Query("INSERT INTO accounts (first_name, last_name, number, password, balance, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		a.FirstName, a.LastName, a.Number, a.Password, a.Balance, a.CreatedAt)

	fmt.Println(resp)
	return err
}

func (s *PostgresStore) Delete(id int) error {
	_, err := s.db.Exec("DELETE FROM accounts WHERE id = $1", id)

	return err
}

func (s *PostgresStore) Update(a *Account) error {
	_, err := s.db.Exec("UPDATE accounts SET first_name = $1, last_name = $2, number = $3, balance = $4, created_at = $5 WHERE id = $6",
		a.FirstName, a.LastName, a.Number, a.Balance, a.CreatedAt, a.ID)

	return err
}

func (s *PostgresStore) Get(id int) (*Account, error) {
	row := s.db.QueryRow("SELECT * FROM accounts WHERE id = $1", id)

	a := &Account{}
	if err := row.Scan(&a.ID, &a.FirstName, &a.LastName, &a.Number, &a.Balance, &a.CreatedAt); err != nil {
		return nil, fmt.Errorf("could not get account: %v", id)
	}

	return a, nil
}

func (s *PostgresStore) GetByNumber(number int64) (*Account, error) {
	row := s.db.QueryRow("SELECT * FROM accounts WHERE number = $1", number)

	a := &Account{}
	if err := row.Scan(&a.ID, &a.FirstName, &a.LastName, &a.Number, &a.Balance, &a.CreatedAt); err != nil {
		return nil, fmt.Errorf("could not get account: %v", number)
	}

	return a, nil
}

func (s *PostgresStore) List() ([]*Account, error) {
	rows, err := s.db.Query("SELECT * FROM accounts")
	if err != nil {
		return nil, fmt.Errorf("could not list accounts: %v", err)
	}

	var accounts []*Account
	for rows.Next() {
		a := &Account{}
		if e := rows.Scan(&a.ID, &a.FirstName, &a.LastName, &a.Number, &a.Balance, &a.CreatedAt); e != nil {
			return nil, e
		}
		accounts = append(accounts, a)
	}

	return accounts, nil
}
