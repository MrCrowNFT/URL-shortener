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
	CHARSET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	SHORTEN_FORMAT = "snap.link/"
)

// Make the database a global variable so that the functions get access to
// already open db
var URLpairDb *sql.DB

type Url_pair struct {
	url   string 
	s_url string 
}

func main() {
	// Open database
	var err error 
	URLpairDb, err = sql.Open("sqlite3", "./URLpair.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	// Close database when finished executing 
	defer URLpairDb.Close()

	// Create table if it doesn't exist
	err = check_table()
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	// Serve static files from the "UI" directory
    fs := http.FileServer(http.Dir("./UI"))
    http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", serveFrontPage)
	http.HandleFunc("/shorten", shortenUrlHandler)
	http.HandleFunc("/s/", redirectHandler)
	
	log.Fatal(http.ListenAndServe(":5500", nil))
}

// Handler function for listening to requests to the root URL ("/") 
// and responds by sending the HTML page to the client
func serveFrontPage(w http.ResponseWriter, r *http.Request){
	// Parse the html file and create the templete
	t, err := template.ParseFiles("./UI/FrontPage.html")
	if err != nil {
		// Return http 500 response if error parsing
		http.Error(w, "Error loading page", http.StatusInternalServerError)
		return
	}
	// Execute the template and write it to the ResponseWriter to 
	// display on the page
	err = t.Execute(w, nil)
	if err != nil {
		http.Error(w, "Error executing page load", http.StatusInternalServerError)
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

	// Shorten the URL 
	s_url , err := shorten(url)
	if err != nil{
		http.Error(w, "Error shortening the Url", http.StatusInternalServerError)
		return
	}

	// Return the shorten url to user as response
	fmt.Fprintf(w, "%s", s_url)
}

// Handles redirect shorten url to the original url
func redirectHandler(w http.ResponseWriter, r *http.Request){
	// Get the shorten url
	s_url := r.URL.Path

	var url string
	// Select the url pair to the shorten url and copy the url into the url variable
	err := URLpairDb.QueryRow(`SELECT 1 FROM url_pairs WHERE s_url = ?`, s_url).Scan(&url)
	if err == sql.ErrNoRows{
		http.NotFound(w, r)
		return
	} else if err != nil{
		http.Error(w, "Error accesing database", http.StatusInternalServerError)
		return
	}

	// Redirect user to the original database
	http.Redirect(w, r, url, http.StatusFound)
}

// Creates table on database if it does not exist
func check_table()(err error) {
	// Create table if not exists
	_, err = URLpairDb.Exec(`CREATE TABLE IF NOT EXISTS url_pairs(
	url TEXT NOT NULL, 
	s_url TEXT NOT NULL);`)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return err
}

// Shorten the url with random 7 alphanumeric characters
func shorten(url string)(string, error){
	var s_url string

	// Check if the url already into the database
	exists, err := check_url(url)
	if err != nil{
		log.Fatal(err)
	}

	// If url already on db return the paired shorten url get the short url from the db table
	if exists{
		query := `SELECT s_url FROM url_pairs WHERE url = ?`
		URLpairDb.QueryRow(query, url).Scan(&s_url)

		return s_url, nil
	} 

	// If is new url, get new random shorten url
	s_url = create_shorten_url(url)

	// Store the new pair in the database
	create_pair(url, s_url)

	return s_url, nil
}

func create_shorten_url (url string)(s_url string){
	for {
		seed := rand.NewSource(time.Now().UnixNano())
		random := rand.New(seed)

		// Make slice of length 7
		result := make([]byte, R_LENGTH) 

		for i := range result{
		// Get pseudo random char from charset 
		result[i] = CHARSET[random.Intn(len(CHARSET))]
		}
		// Concatenate the result with the shorten url format
		s_url = SHORTEN_FORMAT + string(result)

		// Check if the s_url is unique
		unique, err := check_s_url(s_url)
		if err != nil{
		log.Fatal(err)
		}

		// Only continue if the s_url is unique
		if unique {
			break
		}
	
	}

	return s_url
}

// Adds the pair to the database
func create_pair(url string, s_url string){
	// Prepare the statement to add a new pair into the database
	statement, err := URLpairDb.Prepare(`INSERT INTO url_pairs(url, s_url) VALUES(?, ?)`)
	if err != nil{
		log.Fatal(err)
	} 

	// Execute the statement to add the url along with ir shorten version into the database
	_ , err = statement.Exec(url, s_url)
	if err != nil{
		log.Fatalf("Error inserting pair into database : v%", err)
	}
}

// Check if url given has already been saveb in db, in wich case, return the saved shorten url.
// if is new url, shorten it, create pair and add it to the database.
func check_url(url string)(bool, error){
	// Create query to get the shorten url
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM url_pairs WHERE url = ?)`

	// Scans the db and sets the exists variable accordingly
	err := URLpairDb.QueryRow(query, url).Scan(&exists)
	return exists, err
}

// Check if the shorten url was already used
func check_s_url(s_url string)(bool, error){
	// Create query to get the url
	var s_exists bool 
	query := `SELECT EXISTS(SELECT 1 FROM url_pairs WHERE s_url = ?)`

	// Scans the db and sets the exists variable accordingly
	err := URLpairDb.QueryRow(query, s_url).Scan(&s_exists)
	return s_exists, err
}