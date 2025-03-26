package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/Elchin91/GoDashboard/config"
	_ "github.com/go-sql-driver/mysql"
)

var (
	DB *sql.DB
)

// InitDB initializes the database connection
func InitDB(cfg *config.Config) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	
	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	// Set connection pool parameters
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)
	DB.SetConnMaxIdleTime(5 * time.Minute)
	
	// Test the connection
	if err = DB.Ping(); err != nil {
		return err
	}

	log.Println("Database connection established successfully")
	return nil
}

// ExecuteQuery executes a SQL query and returns the results as a slice of maps
func ExecuteQuery(query string, params ...interface{}) ([]map[string]interface{}, error) {
	start := time.Now()
	
	rows, err := DB.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := columns[i]
			// Convert []byte to string if needed
			b, ok := val.([]byte)
			if ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	duration := time.Since(start)
	log.Printf("Query executed in %v: %s", duration, query[:min(100, len(query))])
	
	return results, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}