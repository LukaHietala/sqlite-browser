package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type Templates struct {
    templates *template.Template
}

func (t *Templates) LoadTemplates() {
    t.templates = template.Must(template.ParseGlob("web/views/*.html"))
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
}

func (t *Templates) RenderTemplate(w http.ResponseWriter, name string, data interface{}) {
    t.templates.ExecuteTemplate(w, name, data)
}

var (
    port   = flag.String("port", "8080", "HTTP server port")
    dbPath string
)

type Table struct {
    Name string
}

func main() {
    flag.Parse()

    if len(flag.Args()) < 1 {
        fmt.Println("Error: SQLite db file path required")
        fmt.Println("<database-file>")
        os.Exit(1)
    }

    dbPath = flag.Args()[0]

    err := InitDB(dbPath)
    if err != nil {
        log.Fatalf("Error starting db: %v", err)
    }
    defer CloseDB()

    templates := &Templates{}
    templates.LoadTemplates()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/" {
            http.NotFound(w, r)
            return
        }

        tables, err := GetTables()
        if err != nil {
            http.Error(w, "Error querying tables: "+err.Error(), http.StatusInternalServerError)
            return
        }

        data := struct {
            DbPath string
            Tables []string
        }{
            DbPath: dbPath,
            Tables: tables,
        }

        templates.RenderTemplate(w, "index", data)
    })

    http.HandleFunc("/table/", func(w http.ResponseWriter, r *http.Request) {
        tableName := r.URL.Path[len("/table/"):]
        if tableName == "" {
            http.Redirect(w, r, "/", http.StatusSeeOther)
            return
        }

        exists, err := TableExists(tableName)
        if err != nil {
            http.Error(w, "Error checking table: "+err.Error(), http.StatusInternalServerError)
            return
        }
        if !exists {
            http.Error(w, "Table not found", http.StatusNotFound)
            return
        }

        columns, rows, err := GetTableData(tableName)
        if err != nil {
            http.Error(w, "Error fetching table data: "+err.Error(), http.StatusInternalServerError)
            return
        }

        data := struct {
            TableName string
            Columns   []string
            Rows      [][]string
            Query     string
            QueryTime string
        }{
            TableName: tableName,
            Columns:   columns,
            Rows:      rows,
            Query:     "SELECT * FROM " + tableName + " LIMIT 1000",
        }

        templates.RenderTemplate(w, "title", data)
    })

    http.HandleFunc("POST /query", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }

        query := r.FormValue("query")

        if query == "" {
            http.Redirect(w, r, "/", http.StatusSeeOther)
            return
        }

        cols, rows, timeTaken, err := RunQuery(query)
        if err != nil {
            http.Error(w, "Error running query: "+err.Error(), http.StatusInternalServerError)
            return
        }

        tables, err := GetTables()
        if err != nil {
            http.Error(w, "Error querying tables: "+err.Error(), http.StatusInternalServerError)
            return
        }

        data := struct {
            DbPath    string
            Tables    []string
            Query     string
            QueryTime string
            Rows      [][]string
            Columns   []string
            Error     string
        }{
            DbPath:    dbPath,
            Tables:    tables,
            Query:     query,
            QueryTime: timeTaken,
            Rows:      rows,
            Columns:   cols,
        }

        templates.RenderTemplate(w, "index", data)
    })

    fmt.Printf("Server starting on http://localhost:%s\n", *port)
    fmt.Printf("Database from: %s\n", dbPath)
    log.Fatal(http.ListenAndServe(":"+*port, nil))
}

