package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"strings"

	"github.com/BurntSushi/toml"
)

type Mailer struct {
	conf Conf
}

type Conf struct {
	Addr          string
	EmailSMTPAddr string `toml:"email_smtp_addr"`
	EmailFrom     string `toml:"email_from"`
	EmailPassword string `toml:"email_password"`
	Users         map[string]*User
}

type User struct {
	Email   string
	Filters []*Filter
}

type Filter struct {
	Authors  []string
	Branches []string
	Repos    []string
}

type PushEvent struct {
	Ref        string
	Repository *Repository
	Commits    []*Commit
}

type Repository struct {
	Name string
}

type Commit struct {
	Id     string
	Author *Author
}

type Author struct {
	Name     string
	Email    string
	Username string
}

func main() {
	var confPath = flag.String("conf", "conf.toml", "Path to TOML config")
	flag.Parse()
	var m Mailer
	if _, err := toml.DecodeFile(*confPath, &m.conf); err != nil {
		log.Fatalln("Error parsing conf file:", err)
	}
	log.Printf("Listening on %s", m.conf.Addr)
	log.Fatal(http.ListenAndServe(m.conf.Addr, &m))
}

func (m *Mailer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		if r.Method != "GET" {
			http.Error(w, "method not allowed (expected GET)", http.StatusMethodNotAllowed)
			return
		}
		w.Write([]byte("ghmailer ok\n"))
		return
	case "/push":
		if r.Method != "POST" {
			http.Error(w, "method not allowed (expected POST)", http.StatusMethodNotAllowed)
			return
		}
		m.HandlePush(w, r)
		return
	}
	http.Error(w, "not found", 404)
}

func (m *Mailer) HandlePush(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading post body:", err)
		http.Error(w, "internal error", 500)
		return
	}
	var pushEvent PushEvent
	if err := json.Unmarshal(b, &pushEvent); err != nil {
		log.Printf("Error unmarshaling post body: %s; %s", err, string(b))
		http.Error(w, "internal error", 500)
		return
	}
	for _, user := range m.conf.Users {
		for _, commit := range user.FilterCommits(&pushEvent) {
			if err := m.SendCommitEmail(user, commit); err != nil {
				log.Printf("Error sending commit email: %s", err)
			}
		}
	}
}

func (u *User) FilterCommits(pe *PushEvent) []*Commit {
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
		if len(split) != 3 {
			log.Println("Error parsing ref:", pe.Ref)
			continue
		}
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
	var commits []*Commit
	for _, commit := range pe.Commits {
		if _, ok := shas[commit.Id]; ok {
			commits = append(commits, commit)
		}
	}
	return commits
}

func (m *Mailer) SendCommitEmail(u *User, commit *Commit) error {
	host := strings.SplitN(m.conf.EmailSMTPAddr, ":", 2)[0]
	auth := smtp.PlainAuth(
		"",
		m.conf.EmailFrom,
		m.conf.EmailPassword,
		host,
	)
	// TODO: Flesh out email content
	subject := "New commit: " + commit.Id
	body := commit.Id
	return smtp.SendMail(
		m.conf.EmailSMTPAddr,
		auth,
		m.conf.EmailFrom,
		[]string{u.Email},
		[]byte("Subject: "+subject+"\r\n\r\n"+body),
	)
}
