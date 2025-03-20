package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/tursodatabase/go-libsql"
)

var db *sql.DB

// sqlite_master https://wiki.tcl-lang.org/page/sqlite_master


// LibSql for more functionality

func InitDB(dbPath string) error {
    var err error
    db, err = sql.Open("libsql", "file:"+dbPath+"")
    if err != nil {
        return fmt.Errorf("failed to open database: %v", err)
    }

    err = db.Ping()
    if err != nil {
        return fmt.Errorf("failed to connect to database: %v", err)
    }

    return nil
}

func CloseDB() {
    if db != nil {
        db.Close()
    }
}

func GetTables() ([]string, error) {
    rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var tables []string
    for rows.Next() {
        var tableName string
        if err := rows.Scan(&tableName); err != nil {
            return nil, err
        }
        tables = append(tables, tableName)
    }
    return tables, nil
}

func TableExists(tableName string) (bool, error) {
    var count int
    err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
    if err != nil {
        return false, err
    }
    return count > 0, nil
}

func GetTableData(tableName string) ([]string, [][]string, error) {
    // TODO: Pagination
    rows, err := db.Query(fmt.Sprintf("SELECT * FROM '%s' LIMIT 1000", tableName))
    if err != nil {
        return nil, nil, err
    }
    defer rows.Close()

    columns, err := rows.Columns()
    if err != nil {
        return nil, nil, err
    }

    var resultRows [][]string
    for rows.Next() {
        row := make([]interface{}, len(columns))
        rowPointers := make([]interface{}, len(columns))
        for i := range row {
            rowPointers[i] = &row[i]
        }
        
        if err := rows.Scan(rowPointers...); err != nil {
            return nil, nil, err
        }
        
        stringRow := make([]string, len(columns))
        for i, val := range row {
            stringRow[i] = formatValue(val)
        }
        
        resultRows = append(resultRows, stringRow)
    }

    return columns, resultRows, nil
}

func formatValue(value interface{}) string {
    if value == nil {
        return "NULL"
    }
    return fmt.Sprintf("%v", value)
}

func RunQuery(query string) ([]string, [][]string, string, error) {
    start := time.Now()
    rows, err := db.Query(query)
    if err != nil {
        return nil, nil, "", err
    }
    defer rows.Close()

    columns, err := rows.Columns()
    if err != nil {
        return nil, nil, "", err
    }

    var resultRows [][]string
    for rows.Next() {
        row := make([]interface{}, len(columns))
        rowPointers := make([]interface{}, len(columns))
        for i := range row {
            rowPointers[i] = &row[i]
        }
        
        if err := rows.Scan(rowPointers...); err != nil {
            return nil, nil, "", err
        }
        
        stringRow := make([]string, len(columns))
        for i, val := range row {
            stringRow[i] = formatValue(val)
        }
        
        resultRows = append(resultRows, stringRow)
    }
    
    queryTime := time.Since(start).String()
    return columns, resultRows, queryTime, nil
}