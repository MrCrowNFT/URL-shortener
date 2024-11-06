package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

type Url_pair struct {
	url   string
	s_url string
}

func main() {
	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		t, err := template.ParseFiles("./UI/FrontPage.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		err = t.Execute()
		if err != nil {
			fmt.Println(err)

		}
	})
	log.Fatal(http.ListenAndServe(":5500", nil))
}

// Creates table on database if it does not exist
func init() {
	//open database
	URLpairDb, err := sql.Open("sqlite3", "./URLpair.db")
	if err != nil {
		log.Fatal(err)
	}
	defer URLpairDb.Close()

	//create table if not exists
	_, err = URLpairDb.Exec(`CREATE TABLE IF NOT EXISTS url_pairs(
	url TEXT NOT NULL, 
	url_s TEXT NOT NULL);`)
	if err != nil {
		log.Fatal(err)
	}
}

// Shorten the url with random 7 digit number
func shorten(url string){

}

// Adds the pair to the database
func create_pair(url string, url_s string){

}
