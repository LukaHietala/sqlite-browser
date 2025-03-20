package main

import (
	"database/sql"
	"fmt"
	"strings"
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
    
    // disable foreign key constraints because db.exec is buggy
    if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
        return []string{"Error"}, [][]string{{"Failed to disable foreign key constraints"}}, "", err
    }
    defer func() {
        db.Exec("PRAGMA foreign_keys = ON")
    }()
    
    queries := splitQueries(query)
    
    var finalColumns []string
    var finalRows [][]string
    var lastError error
    var allResults []struct {
        columns []string
        rows    [][]string
    }
    
    for _, singleQuery := range queries {
        singleQuery = strings.TrimSpace(singleQuery)
        if singleQuery == "" {
            continue
        }
        
        if validatePrefix(singleQuery, "CREATE", "ALTER", "DROP", "DELETE", "UPDATE", "INSERT") {
            result, err := db.Exec(singleQuery)
            if err != nil {
                finalColumns = []string{"Error"}
                finalRows = [][]string{{fmt.Sprintf("Error: %v", err)}}
                lastError = err
                break
            }
            
            if validatePrefix(singleQuery, "DELETE", "UPDATE", "INSERT") {
                affected, _ := result.RowsAffected()
                allResults = append(allResults, struct {
                    columns []string
                    rows    [][]string
                }{
                    columns: []string{"Result"},
                    rows:    [][]string{{fmt.Sprintf("%d row(s) affected", affected)}},
                })
            } else {
                allResults = append(allResults, struct {
                    columns []string
                    rows    [][]string
                }{
                    columns: []string{"Result"},
                    rows:    [][]string{{"Query executed successfully"}},
                })
            }
        } else {
            rows, err := db.Query(singleQuery)
            if err != nil {
                finalColumns = []string{"Error"}
                finalRows = [][]string{{fmt.Sprintf("Error: %v", err)}}
                lastError = err
                break
            }
            
            columns, err := rows.Columns()
            if err != nil {
                rows.Close()
                lastError = err
                break
            }
            
            var resultRows [][]string
            for rows.Next() {
                row := make([]interface{}, len(columns))
                rowPointers := make([]interface{}, len(columns))
                for i := range row {
                    rowPointers[i] = &row[i]
                }
                
                if err := rows.Scan(rowPointers...); err != nil {
                    rows.Close()
                    lastError = err
                    break
                }
                
                stringRow := make([]string, len(columns))
                for i, val := range row {
                    stringRow[i] = formatValue(val)
                }
                
                resultRows = append(resultRows, stringRow)
            }
            rows.Close()
            
            if lastError != nil {
                break
            }
            
            allResults = append(allResults, struct {
                columns []string
                rows    [][]string
            }{
                columns: columns,
                rows:    resultRows,
            })
        }
    }
    
    if len(allResults) > 0 {
        lastResult := allResults[len(allResults)-1]
        finalColumns = lastResult.columns
        finalRows = lastResult.rows
    }
    
    queryTime := time.Since(start).String()
    if lastError != nil {
        return finalColumns, finalRows, queryTime, lastError
    }
    
    return finalColumns, finalRows, queryTime, nil
}

func splitQueries(query string) []string {
    var queries []string
    var currentQuery strings.Builder
    inQuote := false
    quoteChar := rune(0)
    
    for _, char := range query {
        if (char == '\'' || char == '"') && (quoteChar == 0 || quoteChar == char) {
            inQuote = !inQuote
            if inQuote {
                quoteChar = char
            } else {
                quoteChar = 0
            }
        }
        
        if char == ';' && !inQuote {
            queries = append(queries, currentQuery.String())
            currentQuery.Reset()
        } else {
            currentQuery.WriteRune(char)
        }
    }
    
    if currentQuery.Len() > 0 {
        queries = append(queries, currentQuery.String())
    }
    
    return queries
}

func validatePrefix(s string, prefixes ...string) bool {
    s = strings.TrimSpace(s)
    for _, prefix := range prefixes {
        if strings.HasPrefix(strings.ToUpper(s), prefix) {
            return true
        }
    }
    return false
}
