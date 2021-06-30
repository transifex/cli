package txlib

import (
	"time"

	"github.com/go-git/go-git/v5"
)

func getGitBranch() string {
	// TODO: Find if git repo is setup on a parent folder
	repo, err := git.PlainOpen(".")
	if err != nil {
		return ""
	} else {
		head, err := repo.Head()
		if err != nil {
			return ""
		} else {
			return head.Name().Short()
		}
	}
}

func getLastCommitDate(path string) time.Time {
	// TODO: check if parent folder is repo
	repo, err := git.PlainOpen(".")
	if err != nil {
		return time.Time{}
	}
	cIter, err := repo.Log(
		&git.LogOptions{FileName: &path, Order: git.LogOrderCommitterTime},
	)
	if err != nil {
		return time.Time{}
	}
	commit, err := cIter.Next()
	if err != nil {
		return time.Time{}
	}
	return commit.Author.When
}
