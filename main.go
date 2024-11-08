package main

import (
	"database/sql"
	"fmt"
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
	http.HandleFunc("POST /shorten", shortenUrlHandler)
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

// Handles shorten url request when user submits a URL
func shortenUrlHandler(w http.ResponseWriter, r *http.Request){
	// Check the request is POST request
	if r.Method != http.MethodPost{
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the form to get the input
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	// Retrive the URL input from the form
	url := r.FormValue("url")
	if url == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Check if table database exists
	err = check_table()
	if err != nil{
		http.Error(w, "Error getting database table", http.StatusInternalServerError)
		return
	}

	// Shorten the URL 
	url_s , err := shorten(url)
	if err != nil{
		http.Error(w, "Error shortening the Url", http.StatusInternalServerError)
		return
	}

	// Return the shorten url to user as response
	fmt.Fprint(w, "%s", url_s)
}

// Handles redirect shorten url to the original url
func redirectHandler(w http.ResponseWriter, r *http.Request){
	// Get the shorten url
	url_s := r.URL.Path

	// Open database
	URLpairDb, err := sql.Open("sqlite3", "./URLpair.db")
	if err != nil{
		http.Error(w, "Error accessing database", http.StatusInternalServerError)
		return
	}

	defer URLpairDb.Close()

	var url string
	// Select the url pair to the shorten url
	query := `SELECT EXISTS(SELECT 1 FROM url_pairs WHERE url_s = ?)`

	// Execute the query and copy the url into the url variable
	URLpairDb.QueryRow(query, url_s).Scan(&url)

	// Redirect user to the original database
	http.Redirect(w, r, url, http.StatusFound)

}

// Creates table on database if it does not exist
func check_table()(err error) {
	//open database
	URLpairDb, err := sql.Open("sqlite3", "./URLpair.db")
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer URLpairDb.Close()

	// Create table if not exists
	_, err = URLpairDb.Exec(`CREATE TABLE IF NOT EXISTS url_pairs(
	url TEXT NOT NULL, 
	url_s TEXT NOT NULL);`)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
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
	
	// Create query to get the shorten url
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

	// Prepare the statement to add a new pair into the database
	statement, err := URLpairDb.Prepare(`INSERT INTO url_pairs(url, url_s) VALUES(?, ?)`)
	if err != nil{
		log.Fatal(err)
	} 

	// Execute the statement to add the url along with ir shorten version into the database
	statement.Exec(url, url_s)
}

