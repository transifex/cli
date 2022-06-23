package txlib

import (
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
)

func getGitBranch() string {
	result := getGitBranchFromBinary()
	if result != "" {
		return result
	}
	return getGitBranchFromGoGit()
}

func getGitBranchFromBinary() string {
	out, err := exec.Command("git", "symbolic-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func getGitBranchFromGoGit() string {
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
	result := getLastCommitDateFromBinary(path)
	if result != (time.Time{}) {
		return result
	}
	return getLastCommitDateFromGoGit(path)
}

func getLastCommitDateFromBinary(path string) time.Time {
	outBytes, err := exec.Command(
		"git", "log", "--max-count=1", "--format=format:%at", "--", path,
	).Output()
	if err != nil {
		return time.Time{}
	}
	outInt, err := strconv.Atoi(strings.TrimSpace(string(outBytes)))
	if err != nil {
		return time.Time{}
	}
	return time.Unix(int64(outInt), 0)
}

func getLastCommitDateFromGoGit(path string) time.Time {
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
