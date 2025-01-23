package sqlite

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

// NewSQLiteDB creates a new instance of a sqlite database
func NewSQLiteDB(path string) (*SQLiteDB, error) {
	// Using in-memory databases for testing, special filename, :memory:
	db := &SQLiteDB{
		Drive: "sqlite",
		Path:  path,
	}
	err := db.InitDatabase()
	return db, err
}

// InitDatabase creates the database and table
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
			role INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
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

// Fetch retrieves a list of all users
func (db *SQLiteDB) Fetch(ctx context.Context) ([]domain.User, error) {
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

// GetByID retrieves a user by ID
func (db *SQLiteDB) GetById(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row := db.DB.QueryRowContext(
		context.Background(),
		`SELECT * FROM users WHERE id=$1`, id,
	)
	var user domain.User
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role)
	return &user, err
}

// GetByEmail retrieves a user by email
func (db *SQLiteDB) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := db.DB.QueryRowContext(
		context.Background(),
		`SELECT * FROM users WHERE email=$1`, email,
	)
	var user domain.User
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role)
	return &user, err
}

// GetByUsername retrieves a user by username
func (db *SQLiteDB) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	row := db.DB.QueryRowContext(
		context.Background(),
		`SELECT * FROM users WHERE username=$1`, username,
	)
	var user domain.User
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role)
	return &user, err
}

// Create a new user
func (db *SQLiteDB) Create(ctx context.Context, user *domain.User) error {
	_, err := db.DB.ExecContext(
		context.Background(),
		`INSERT INTO users (id, username, password, email, role) VALUES (?,?,?,?,?);`,
		user.ID, user.Username, user.Password, user.Email, user.Role,
	)
	return err
}

// Update a user by ID
func (db *SQLiteDB) Update(ctx context.Context, user *domain.User) error {
	_, err := db.DB.ExecContext(
		context.Background(),
		`UPDATE users SET username=?, password=?, email=?, role=? WHERE id=?;`,
		user.Username, user.Password, user.Email, user.Role, user.ID,
	)
	return err
}

// Delete a user by ID
func (db *SQLiteDB) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := db.DB.ExecContext(
		context.Background(),
		`DELETE FROM users WHERE id=?;`,
		id,
	)
	return err
}
