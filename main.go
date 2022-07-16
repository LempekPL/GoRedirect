package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	blocked = []string{"admin", "lempek", "lk", "lmpk", "create_redirect", "delete_redirect", "modify_redirect", "get"}
	auth    string
)

const DefaultLink = "https://lmpk.tk"

// Success is bool type
// Description is string explanation of error
type responseCode struct {
	Success     bool   `json:"success"`
	Description string `json:"description"`
}

var (
	CodeOK            = responseCode{true, "Done"}
	CodeNoAuth        = responseCode{false, "No Auth header provided"}
	CodeWrongAuth     = responseCode{false, "Wrong Auth header"}
	CodeNameNotFound  = responseCode{false, "Name not found"}
	CodeNoName        = responseCode{false, "No name provided"}
	CodeNoLink        = responseCode{false, "No link provided"}
	CodeNameBlocked   = responseCode{false, "Name is blocked"}
	CodeNameWrong     = responseCode{false, "Name is not compliant with the pattern. Name pattern: ^[a-zA-Z0-9_.-]*$"}
	CodeLinkWrong     = responseCode{false, "Link is not compliant with the pattern. Link pattern (only https): (https:\\/\\/)([\\da-z\\.-]+)\\.([a-z]{2,6})([\\/\\w\\.-]*)*\\/?"}
	CodeNameLinkWrong = responseCode{false, "Name and link are not compliant with the pattern. Name pattern: ^[a-zA-Z0-9_.-]*$, Link pattern (only https): (https:\\/\\/)([\\da-z\\.-]+)\\.([a-z]{2,6})([\\/\\w\\.-]*)*\\/?"}
	CodeAlreadyExist  = responseCode{false, "Name already exist"}
)

func main() {
	env, err := ioutil.ReadFile(".env")
	if err != nil {
		log.Fatalln(err)
	}
	auth = strings.Split(string(env), "=")[1]
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/create", create_redirect)
	http.HandleFunc("/delete", delete_redirect)
	http.HandleFunc("/modify", modify_redirect)
	http.HandleFunc("/", redirector)

	srv := &http.Server{
		Addr:         "localhost:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}

func create_redirect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	name := r.URL.Query().Get("name")
	link := r.URL.Query().Get("link")
	pass := r.Header.Get("Auth")
	var err error
	if pass == "" {
		err = json.NewEncoder(w).Encode(CodeNoAuth)
	} else if pass != auth {
		err = json.NewEncoder(w).Encode(CodeWrongAuth)
	} else if name == "" {
		err = json.NewEncoder(w).Encode(CodeNoName)
	} else if link == "" {
		err = json.NewEncoder(w).Encode(CodeNoLink)
	} else {
		resp := createRedirect(name, link)
		responseHandler(w, resp)
	}
	if err != nil {
		return
	}
}

func delete_redirect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	name := r.URL.Query().Get("name")
	pass := r.Header.Get("Auth")
	var err error
	if pass == "" {
		err = json.NewEncoder(w).Encode(CodeNoAuth)
	} else if pass != auth {
		err = json.NewEncoder(w).Encode(CodeWrongAuth)
	} else if name == "" {
		err = json.NewEncoder(w).Encode(CodeNoName)
	} else {
		resp := deleteRedirect(name)
		responseHandler(w, resp)
	}
	if err != nil {
		return
	}
}

func modify_redirect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	name := r.URL.Query().Get("name")
	link := r.URL.Query().Get("link")
	pass := r.Header.Get("Auth")
	var err error
	if pass == "" {
		err = json.NewEncoder(w).Encode(CodeNoAuth)
	} else if pass != auth {
		err = json.NewEncoder(w).Encode(CodeWrongAuth)
	} else if name == "" {
		err = json.NewEncoder(w).Encode(CodeNoName)
	} else if link == "" {
		err = json.NewEncoder(w).Encode(CodeNoLink)
	} else {
		prevLink := getRedirect(name)
		resp := deleteRedirect(name)
		if resp == "deleted-redirect" {
			resp2 := createRedirect(name, link)
			if resp2 != "created-redirect" {
				createRedirect(name, prevLink)
			}
			responseHandler(w, resp2)
		} else {
			responseHandler(w, resp)
		}
	}
	if err != nil {
		return
	}
}

func createRedirect(name, link string) string {
	match1, _ := regexp.MatchString("^[a-zA-Z0-9_.-]*$", name)
	match2, _ := regexp.MatchString("(https://)([\\da-z.-]+)\\.([a-z]{2,6})([/\\w.-]*)*/?", link)
	if contains(blocked, name) {
		return "name-is-blocked"
	} else if getRedirect(name) != "" {
		return "name-already-exist"
	} else if !match1 && match2 {
		return "name-not-accepted"
	} else if match1 && !match2 {
		return "link-not-accepted"
	} else if !match1 && !match2 {
		return "name-link-not-accepted"
	}
	input, err := ioutil.ReadFile("redirects.txt")
	if err != nil {
		log.Fatalln(err)
	}
	lines := strings.Split(string(input), "\n")

	s := fmt.Sprintf("%v > %v", name, link)
	lines = append(lines, s)

	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile("redirects.txt", []byte(output), 0644)
	if err != nil {
		log.Fatalln(err)
	}
	return "created-redirect"
}

func deleteRedirect(name string) string {
	input, err := ioutil.ReadFile("redirects.txt")
	if err != nil {
		log.Fatalln(err)
	}

	resp := "name-not-found"
	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, name) {
			lines = append(lines[:i], lines[i+1:]...)
			resp = "deleted-redirect"
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile("redirects.txt", []byte(output), 0644)
	if err != nil {
		log.Fatalln(err)
	}
	return resp
}

func getRedirect(name string) string {
	input, err := ioutil.ReadFile("redirects.txt")
	if err != nil {
		log.Fatalln(err)
	}
	splitFile := strings.Split(string(input), "\n")
	for _, words := range splitFile {
		splitWords := strings.Split(words, " > ")
		if splitWords[0] == name && !contains(blocked, name) {
			return splitWords[1]
		}
	}
	return ""
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func redirector(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	vars := strings.Split(path, "/")
	if len(vars[1]) < 1 {
		// rendering default html if redirect name is not present
		parsedTemplate, _ := template.ParseFiles("index.html")
		err := parsedTemplate.Execute(w, nil)
		if err != nil {
			log.Println("Error executing template :", err)
			return
		}
	} else {
		link := getRedirect(vars[1])
		if link == "" {
			http.Redirect(w, r, DefaultLink, http.StatusTemporaryRedirect)
		}
		http.Redirect(w, r, link, http.StatusTemporaryRedirect)
	}
}

func responseHandler(w http.ResponseWriter, response string) {
	var err error
	switch response {
	case "created-redirect":
		err = json.NewEncoder(w).Encode(CodeOK)
		break
	case "deleted-redirect":
		err = json.NewEncoder(w).Encode(CodeOK)
		break
	case "name-is-blocked":
		err = json.NewEncoder(w).Encode(CodeNameBlocked)
		break
	case "name-already-exist":
		err = json.NewEncoder(w).Encode(CodeAlreadyExist)
		break
	case "name-not-accepted":
		err = json.NewEncoder(w).Encode(CodeNameWrong)
		break
	case "link-not-accepted":
		err = json.NewEncoder(w).Encode(CodeLinkWrong)
		break
	case "name-link-not-accepted":
		err = json.NewEncoder(w).Encode(CodeNameLinkWrong)
		break
	case "name-not-found":
		err = json.NewEncoder(w).Encode(CodeNameNotFound)
		break
	}
	if err != nil {
		log.Fatalln(err)
	}
}
