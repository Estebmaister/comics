package repository

import (
	"comics/domain"
	"context"
	"database/sql"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type SQLiteDB struct {
	DB    *sql.DB
	Drive string
	Path  string
}

func NewSQLiteDB(path string) (*SQLiteDB, error) {
	// Using in-memory databases for testing, special filename, :memory:
	db := &SQLiteDB{
		Drive: "sqlite",
		Path:  path,
	}
	err := db.InitDatabase()
	return db, err
}

func (db *SQLiteDB) InitDatabase() error {
	var err error
	db.DB, err = sql.Open(db.Drive, db.Path)
	if err != nil {
		return err
	}
	_, err = db.DB.ExecContext(
		context.Background(),
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(128) PRIMARY KEY, 
			username TEXT NOT NULL,
			password TEXT NOT NULL, 
			email VARCHAR(255),
			role INTEGER NOT NULL
		)`,
	)
	if err != nil {
		return err
	}
	return nil
}

func (db *SQLiteDB) Close() error {
	return db.DB.Close()
}

func (db *SQLiteDB) FindAll(ctx context.Context) ([]domain.User, error) {
	// TODO: Implement pagination
	rows, err := db.DB.QueryContext(context.Background(), `SELECT * FROM users`)
	if err != nil {
		return nil, err
	}
	var users []domain.User
	for rows.Next() {
		var user domain.User
		err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (db *SQLiteDB) FindById(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row := db.DB.QueryRowContext(
		context.Background(),
		`SELECT * FROM users WHERE id=$1`, id,
	)
	var user domain.User
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role)
	return &user, err
}

func (db *SQLiteDB) Create(ctx context.Context, user *domain.User) error {
	_, err := db.DB.ExecContext(
		context.Background(),
		`INSERT INTO users (id, username, password, email, role) VALUES (?,?,?,?,?);`,
		user.ID, user.Username, user.Password, user.Email, user.Role,
	)
	return err
}

func (db *SQLiteDB) Update(ctx context.Context, user *domain.User) error {
	_, err := db.DB.ExecContext(
		context.Background(),
		`UPDATE users SET username=?, password=?, email=?, role=? WHERE id=?;`,
		user.Username, user.Password, user.Email, user.Role, user.ID,
	)
	return err
}

func (db *SQLiteDB) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := db.DB.ExecContext(
		context.Background(),
		`DELETE FROM users WHERE id=?;`,
		id,
	)
	return err
}
