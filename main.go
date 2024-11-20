package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"math/rand"
	"time"
	"os"
	"os/signal"
	"syscall"
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
	var err error 

	// Open database
	URLpairDb, err = sql.Open("sqlite3", "./URLpair.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	// Verify database connection 
	err = URLpairDb.Ping()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection verified successfully.")

	// Check if table exist
	err = check_table()
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	// Serve static files from the UI folder
    http.Handle("/UI/", http.StripPrefix("/UI/", http.FileServer(http.Dir("./UI"))))

	// Define HTTP handlers
	http.HandleFunc("/", serveFrontPage)
	http.HandleFunc("/shorten", shortenUrlHandler)
	http.HandleFunc("/s/", redirectHandler)

	// Graceful shutdown handling
	// Create a chanel for signals
	stop := make(chan os.Signal, 1)
	// Register the Signals to Listen For
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)


	go func() {
		log.Fatal(http.ListenAndServe(":5500", nil))
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down server...")

	// Close database before exiting
	if err := URLpairDb.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	} else {
		log.Println("Database connection closed successfully.")
	}
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
	log.Println("Received request to shorten URL")
	log.Println("Request method:", r.Method)

	// Check the request is POST request
	if r.Method != http.MethodPost{
		log.Println("Invalid request method:", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the form to get the input
	err := r.ParseForm()
	if err != nil {
		log.Println("Error parsing form:", err)
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}
	log.Println("Form data:", r.Form)

	// Retrive the URL input from the form
	url := r.FormValue("url")
	if url == "" {
		log.Println("No URL provided")
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}
	log.Println("URL received:", url)

	// Shorten the URL 
	s_url , err := shorten(url)
	if err != nil{
		log.Println("Error in shorten function:", err)
		http.Error(w, "Error shortening the Url", http.StatusInternalServerError)
		return
	}

	log.Printf("Shortened URL created: %s", s_url)

	// Return the shorten url to user as response
	fmt.Fprintf(w, "%s", s_url)
}

// Handles redirect shorten url to the original url
func redirectHandler(w http.ResponseWriter, r *http.Request){
	// Get the shorten url
	s_url := r.URL.Path[len("/s/"):]

	var url string
	// Select the url pair to the shorten url and copy the url into the url variable
	err := URLpairDb.QueryRow(`SELECT url FROM url_pairs WHERE s_url = ?`, s_url).Scan(&url)
	if err == sql.ErrNoRows{
		http.NotFound(w, r)
		return
	} else if err != nil{
		http.Error(w, "Error accesing database", http.StatusInternalServerError)
		return
	}

	// Redirect user to the original URL
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
		log.Println("check_url error:", err)
		return "", err
	}
	log.Println("URL exists in DB:", exists)

	// If url already on db return the paired shorten url get the short url from the db table
	if exists == true{
		query := `SELECT s_url FROM url_pairs WHERE url = ?`
		err := URLpairDb.QueryRow(query, url).Scan(&s_url)
		if err != nil{
			if err == sql.ErrNoRows {
				log.Println("URL not found in database.")
			} else {
				log.Println("Database query error:", err)
			}
			return "", err
		}
		return s_url, nil
	} 

	// If is new url, get new random shorten url
	s_url = create_shorten_url()

	// Store the new pair in the database
	err = create_pair(url, s_url)
	if err != nil {
		return "", err
	}

	return s_url, nil
}

func create_shorten_url()(s_url string){
	log.Println("Shortening URL")

	for {
		log.Println("Getting Random Seed")
		seed := rand.NewSource(time.Now().UnixNano())
		random := rand.New(seed)

		// Make slice of length 7
		result := make([]byte, R_LENGTH) 

		for i := range result{
		// Get pseudo random char from charset 
		result[i] = CHARSET[random.Intn(len(CHARSET))]
		}
		// Concatenate the result with the shorten url format
		log.Println("Creating shorten URL")
		s_url = SHORTEN_FORMAT + string(result)

		// Check if the s_url is unique
		log.Println("Checking shorten URL collision")
		unique, err := check_s_url(s_url)
		if err != nil{
			log.Fatal(err)
		}

		// Only continue if the s_url is unique
		log.Println("Checking shorten URL")
		if unique == true{
			break
		}
	
	}

	log.Printf("Generated short URL candidate: %s", s_url)

	return s_url
}

// Adds the pair to the database
func create_pair(url string, s_url string)(err error){
	log.Printf("Attempting to insert pair: %s -> %s", url, s_url)
	
	// Prepare the statement to add a new pair into the database
	statement, err := URLpairDb.Prepare(`INSERT INTO url_pairs(url, s_url) VALUES(?, ?)`)
	if err != nil{
		log.Println("Error preparing statement:", err)
		log.Fatal(err)
	} else {
		log.Printf("Successfully Prepared pair: insertion %s -> %s", url, s_url)
	}
	defer statement.Close()

	// Execute the statement to add the url along with ir shorten version into the database
	_ , err = statement.Exec(url, s_url)
	if err != nil{
		log.Println("Error inserting pair into database:", err)
	} else {
        log.Printf("Successfully inserted pair: %s -> %s", url, s_url)
    }

	return err
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
	var unique bool 
	query := `SELECT EXISTS(SELECT 1 FROM url_pairs WHERE s_url = ?)`

	// Scans the db and sets the exists variable accordingly
	err := URLpairDb.QueryRow(query, s_url).Scan(&unique)
	return !unique, err
}