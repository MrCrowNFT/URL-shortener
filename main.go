package main

import (
	"database/sql"
	//"fmt"
	"html/template"
	"log"
	"net/http"
	"math/rand"
	"time"
	_ "github.com/mattn/go-sqlite3"
)

const (
	R_LENGTH = 7
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type Url_pair struct {
	url   string
	s_url string
}

func main() {
	http.HandleFunc("/", serveFrontPage)
	http.HandleFunc("POST /shorten", shortenUrlHandles)
	http.HandleFunc("/", redirectHandler)
	log.Fatal(http.ListenAndServe(":5500", nil))
}

// Handler function for listening to requests to the root URL ("/") 
// and responds by sending the HTML page to the client
func serveFrontPage(w http.ResponseWriter, r *http.Request){
	// Parse the html file and create the templete
	t, err := template.ParseFiles("./UI/FrontPage.html")
	if err != nil {
		// Return http 500 response if error parsing
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Execute the template and write it to the ResponseWriter to 
	// display on the page
	err = t.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func shortenUrlHandles(w http.ResponseWriter, r *http.Request){

}

func redirectHandler(w http.ResponseWriter, r *http.Request){

}

// Creates table on database if it does not exist
func check_table() {
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

// Shorten the url with random 7 alphanumeric characters
func shorten(url string)(string, error){
	exists, err := check_url(url)
	if err != nil{
		log.Fatal(err)
	}

	// If is new url, return new random 
	if exists == false{
		seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)

	// Make slice of length 7
	result := make([]byte, R_LENGTH) 

	for i := range result{
		// Get pseudo random char from charset 
		result[i] = charset[random.Intn(len(charset))]
	}
	url_s := string(result)
	// Store the new pair in the database
	create_pair(url, url_s)
	return url_s, nil
	} 

	// If url already on db return the paired shorten url
	// Open db
	URLpairDb, err := sql.Open("sqlite3", "./URLpair.db")
	if err != nil {
		log.Fatal(err)
	}
	// Close db when finishing executing
	defer URLpairDb.Close()

	var url_s string

	// Get the short url from the db table
	query := `SELECT url_s FROM url_pairs WHERE url = ?`
	URLpairDb.QueryRow(query, url).Scan(&url_s)

	return url_s, nil
}

// Check if url given has already been saveb in db, in wich case, return the saved shorten url.
// if is new url, shorten it, create pair and add it to the database.
func check_url(url string)(bool, error){
	// Open database
	URLpairDb, err := sql.Open("sqlite3", "./URLpair.db")
	if err != nil {
		log.Fatal(err)
	}
	// Close database when finished executing 
	defer URLpairDb.Close()

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM url_pairs WHERE url = ?)`

	// Scans the db and sets the exists variable accordingly
	URLpairDb.QueryRow(query, url).Scan(&exists)

	return exists, nil
}


// Adds the pair to the database
func create_pair(url string, url_s string){
	// Open database
	URLpairDb, err := sql.Open("sqlite3", "./URLpair.db")
	if err != nil {
		log.Fatal(err)
	}
	// Close database when finished executing 
	defer URLpairDb.Close()

	statement, err := URLpairDb.Prepare(`INSERT INTO url_pairs(url, url_s) VALUES(?, ?)`)
	if err != nil{
		log.Fatal(err)
	} 

	statement.Exec(url, url_s)
}
