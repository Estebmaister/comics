package sql

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func mysqlConn() {
	// Load .env variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Open connection with MySQL DB
	db, err := sql.Open("mysql", os.Getenv("DATABASE_DSN"))
	if err != nil {
		log.Fatalf("Error opening database: %v\n", err)
	}
	defer db.Close()

	// Ensure that the connection works
	err = db.Ping()
	if err != nil {
		log.Fatalf("Error connecting database: %v\n", err)
	}

	fmt.Println("Connected to database")

	// Execute the SHOW TABLES query to list all tables in the database
	tables, err := db.Query("SHOW TABLES")
	if err != nil {
		log.Fatalf("Failed to execute SHOW TABLES query: %v\n", err)
	}
	defer tables.Close()

	fmt.Println("Database structure:")

	for tables.Next() {
		var tableName string
		if err := tables.Scan(&tableName); err != nil {
			log.Fatalf("Failed to scan table name: %v\n", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.Debug)

		fmt.Printf("\n[ Table: %s ]\n\n", tableName)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t\n", "Field", "Type", "Null", "Key", "Default", "Extra")

		// Get the structure of the current table
		structureQuery := fmt.Sprintf("DESCRIBE %s", tableName)
		columns, err := db.Query(structureQuery)
		if err != nil {
			log.Fatalf("Failed to describe table %s: %v\n", tableName, err)
		}
		defer columns.Close()

		for columns.Next() {
			var field, colType, null, key, defaultVal, extra sql.NullString
			err := columns.Scan(&field, &colType, &null, &key, &defaultVal, &extra)
			if err != nil {
				log.Fatalf("Failed to scan column: %v\n", err)
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t\n",
				field.String, colType.String, null.String, key.String, defaultVal.String, extra.String)
		}

		w.Flush()
	}
}
