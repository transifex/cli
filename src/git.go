package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
)

func getGitDir(projectDir string) (string, error) {
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	gitDir := ""
	currentDir := projectDir
	for {
		rel, err := filepath.Rel(currentWorkingDir, currentDir)
		if err != nil {
			return "", err
		}
		if strings.Contains(rel, "..") {
			return "", err
		}
		checkDir, err := os.Stat(filepath.Join(currentDir, ".git"))
		if os.IsNotExist(err) || !checkDir.IsDir() {
			currentDir = filepath.Dir(currentDir)
		} else {
			gitDir = currentDir
			break
		}
	}

	return gitDir, nil
}

func getGitBranch(gitDir string) (string, error) {
	gitBranch := ""
	repo, err := git.PlainOpen(gitDir)
	if err != nil {
		return gitBranch, err
	}

	head, err := repo.Head()
	if err != nil {
		return gitBranch, err
	}

	return head.Name().Short(), nil
}

func lastCommitDate(gitDir string, fileName string) (time.Time, error) {
	var commitDate time.Time

	repo, err := git.PlainOpen(gitDir)
	if err != nil {
		return commitDate, err
	}

	cIter, _ := repo.Log(&git.LogOptions{FileName: &fileName, Order: git.LogOrderCommitterTime})
	if err != nil {
		return commitDate, err
	}
	commit, err := cIter.Next()
	if err != nil {
		return commitDate, err
	}
	return commit.Author.When, nil
}
