package sqlite

import (
	"comics/domain"
	"context"
	"database/sql"

	"github.com/google/uuid"
	_ "modernc.org/sqlite" // SQLite driver
)

// UserRepo represents a connection to a SQLite database
// DB: The database connection
// Drive: The database driver
// Path: The path to the database file
type UserRepo struct {
	DB    *sql.DB
	Drive string
	Path  string
}

// NewSQLiteUserRepo creates a new instance of a sqlite database
func NewSQLiteUserRepo(path string) (*UserRepo, error) {
	// Using in-memory databases for testing, special filename, :memory:
	db := &UserRepo{
		Drive: "sqlite",
		Path:  path,
	}
	err := db.InitDatabase()
	return db, err
}

// InitDatabase creates the database and table
func (db *UserRepo) InitDatabase() error {
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

// Close closes the database connection
func (db *UserRepo) Close() error {
	return db.DB.Close()
}

// Fetch retrieves a list of all users
func (db *UserRepo) Fetch(ctx context.Context) ([]domain.User, error) {
	rows, err := db.DB.QueryContext(ctx, `SELECT * FROM users`)
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
func (db *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row := db.DB.QueryRowContext(ctx,
		`SELECT * FROM users WHERE id=$1`, id,
	)
	var user domain.User
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role)
	return &user, err
}

// GetByEmail retrieves a user by email
func (db *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := db.DB.QueryRowContext(ctx,
		`SELECT * FROM users WHERE email=$1`, email,
	)
	var user domain.User
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role)
	return &user, err
}

// GetByUsername retrieves a user by username
func (db *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	row := db.DB.QueryRowContext(ctx,
		`SELECT * FROM users WHERE username=$1`, username,
	)
	var user domain.User
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.Role)
	return &user, err
}

// Create a new user
func (db *UserRepo) Create(ctx context.Context, user *domain.User) error {
	_, err := db.DB.ExecContext(ctx,
		`INSERT INTO users (id, username, password, email, role) VALUES (?,?,?,?,?);`,
		user.ID, user.Username, user.Password, user.Email, user.Role,
	)
	return err
}

// Update a user by ID
func (db *UserRepo) Update(ctx context.Context, user *domain.User) error {
	_, err := db.DB.ExecContext(ctx,
		`UPDATE users SET username=?, password=?, email=?, role=? WHERE id=?;`,
		user.Username, user.Password, user.Email, user.Role, user.ID,
	)
	return err
}

// Delete a user by ID
func (db *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := db.DB.ExecContext(ctx,
		`DELETE FROM users WHERE id=?;`,
		id,
	)
	return err
}
