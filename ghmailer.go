package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/BurntSushi/toml"
)

var confFlag = flag.String("conf", "conf.toml", "Path to TOML config")
var conf Conf

type Filter struct {
	Authors  []string
	Branches []string
	Repos    []string
}

type User struct {
	Email   string
	Filters []Filter
}

type Conf struct {
	Addr  string
	Users map[string]User
}

type Author struct {
	Name     string
	Email    string
	Username string
}

type Commit struct {
	Id     string
	Author Author
}

type Repository struct {
	Name string
}

type PushEvent struct {
	Ref        string
	Repository Repository
	Commits    []Commit
}

func (u User) FilterCommits(pe PushEvent) []Commit {
	shas := make(map[string]struct{})

	for _, filter := range u.Filters {
		// Check repos
		repoMatched := len(filter.Repos) == 0
		for _, repo := range filter.Repos {
			if repo == pe.Repository.Name {
				repoMatched = true
			}
		}
		if !repoMatched {
			continue
		}

		// Check branches
		split := strings.SplitN(pe.Ref, "/", 3)
		pushedBranch := split[len(split)-1]
		branchMatched := len(filter.Branches) == 0
		for _, branch := range filter.Branches {
			if branch == pushedBranch {
				branchMatched = true
			}
		}
		if !branchMatched {
			continue
		}

		// Check authors
		for _, commit := range pe.Commits {
			if len(filter.Authors) == 0 {
				shas[commit.Id] = struct{}{}
			}
			for _, email := range filter.Authors {
				if email == commit.Author.Email {
					shas[commit.Id] = struct{}{}
				}
			}
		}
	}

	// Collect and return commits
	var commits []Commit
	for _, commit := range pe.Commits {
		if _, ok := shas[commit.Id]; ok {
			commits = append(commits, commit)
		}
	}
	return commits
}

func logRequest(r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL.Path)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	if r.Method != "GET" || r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Not Found")
		return
	}
	fmt.Fprintf(w, "OK\n")
}

func pushHandler(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	if r.Method != "POST" {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Not Found")
		return
	}
	jsonData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var pushEvent PushEvent
	err = json.Unmarshal(jsonData, &pushEvent)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, user := range conf.Users {
		commits := user.FilterCommits(pushEvent)
		// TODO: Send commit emails
		for _, commit := range commits {
			log.Printf("Send email to %s for %s %s %s\n",
				user.Email, pushEvent.Repository.Name, pushEvent.Ref, commit.Id)
		}
	}
}

func parseConf() error {
	_, err := toml.DecodeFile(*confFlag, &conf)
	return err
}

func main() {
	flag.Parse()
	err := parseConf()
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/push", pushHandler)

	log.Printf("Listening on %s", conf.Addr)
	log.Fatal(http.ListenAndServe(conf.Addr, nil))
}