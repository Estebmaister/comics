package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type User struct {
	ID            int
	Name          string
	Email         string
	Relationships []string
}

func connectPostgres(connStr string) *sql.DB {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func connectNeo4j(uri, username, password string) neo4j.DriverWithContext {
	driver, err := neo4j.NewDriverWithContext(
		uri,
		neo4j.BasicAuth(username, password, ""),
	)
	if err != nil {
		log.Fatal(err)
	}

	return driver
}

func extractUsersFromPostgres(db *sql.DB) []User {
	query := `SELECT id, name, email FROM users`
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			log.Fatal(err)
		}
		users = append(users, u)
	}

	return users
}

func populateNeo4j(ctx context.Context, driver neo4j.DriverWithContext, users []User) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	for _, user := range users {
		_, err := session.Run(ctx,
			"CREATE (u:User {id: $id, name: $name, email: $email})",
			map[string]any{
				"id":    user.ID,
				"name":  user.Name,
				"email": user.Email,
			},
		)
		if err != nil {
			log.Printf("Error creating node for user %s: %v", user.Name, err)
		}
	}
}

func main() {
	ctx := context.WithoutCancel(context.Background())
	// PostgreSQL Connection
	postgresConnStr := "postgres://username:password@localhost/dbname?sslmode=disable"
	pgDB := connectPostgres(postgresConnStr)
	defer pgDB.Close()

	// Neo4j Connection
	neo4jUri := "neo4j://localhost:7687"
	neo4jUsername := "neo4j"
	neo4jPassword := "your_password"
	neo4jDriver := connectNeo4j(neo4jUri, neo4jUsername, neo4jPassword)
	defer neo4jDriver.Close(ctx)

	// Extract data from Postgres
	users := extractUsersFromPostgres(pgDB)

	// Populate Neo4j
	populateNeo4j(ctx, neo4jDriver, users)

	fmt.Println("Data migration completed successfully!")
}
