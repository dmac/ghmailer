package main

import "testing"

var userMock = User{
	Email: "user@example.com",
	Filters: []Filter{
		Filter{
			Authors:  []string{"alice@example.com"},
			Branches: []string{"master"},
			Repos:    []string{"public-repo"},
		},
	},
}

var pushEvent = PushEvent{
	Ref:        "refs/heads/master",
	Repository: Repository{Name: "public-repo"},
	Commits: []Commit{
		Commit{
			Id: "abc123",
			Author: Author{
				Email: "alice@example.com",
			},
		},
	},
}

func TestFilterCommitsEmpty(t *testing.T) {
	user := userMock
	user.Filters[0].Authors = []string{}
	user.Filters[0].Branches = []string{}
	user.Filters[0].Repos = []string{}
	commits := user.FilterCommits(pushEvent)
	if len(commits) < 1 {
		t.Error("Commits missing when filtering with empty criteria")
	}

	user.Filters = []Filter{}
	commits = user.FilterCommits(pushEvent)
	if len(commits) > 0 {
		t.Error("Commits returned when no filters defined")
	}
}

func TestFilterCommitsByRepo(t *testing.T) {
	user := userMock
	user.Filters[0].Authors = []string{}
	user.Filters[0].Branches = []string{}
	user.Filters[0].Repos = []string{"public-repo", "private-repo"}
	commits := user.FilterCommits(pushEvent)
	if len(commits) < 1 {
		t.Error("Commit missing when filtering by repo")
	}
	user.Filters[0].Repos = []string{"private-repo"}
	commits = user.FilterCommits(pushEvent)
	if len(commits) > 0 {
		t.Error("Extra commit when filtering by repo")
	}
}

func TestFilterCommitsByBranch(t *testing.T) {
	user := userMock
	user.Filters[0].Authors = []string{}
	user.Filters[0].Branches = []string{"master", "feature"}
	user.Filters[0].Repos = []string{}
	commits := user.FilterCommits(pushEvent)
	if len(commits) < 1 {
		t.Error("Commit missing when filtering by branch")
	}
	user.Filters[0].Branches = []string{"feature"}
	commits = user.FilterCommits(pushEvent)
	if len(commits) > 0 {
		t.Error("Extra commit when filtering by branch")
	}
}

func TestFilterCommitsByAuthor(t *testing.T) {
	user := userMock
	user.Filters[0].Authors = []string{"alice@example.com", "bob@example.com"}
	user.Filters[0].Branches = []string{}
	user.Filters[0].Repos = []string{}
	commits := user.FilterCommits(pushEvent)
	if len(commits) < 1 {
		t.Error("Commit missing when filtering by author")
	}
	user.Filters[0].Authors = []string{"bob@example.com"}
	commits = user.FilterCommits(pushEvent)
	if len(commits) > 0 {
		t.Error("Extra commit when filtering by author")
	}
}
