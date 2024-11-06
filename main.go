package main

import (
	"fmt"
	"go/constant"
	"html/template"
	"log"
	"net/http"
)

type Url_pair struct {
	url string;
	s_url string;
}

 func main(){
	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		t , err := template.ParseFiles("./UI/FrontPage.html")
		if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		err = t.Execute()
		if err != nil{
			fmt.Println(err)
		
		}
	})
	log.Fatal(http.ListenAndServe(":5500", nil))
 }