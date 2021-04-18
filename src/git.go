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

func lastCommitDate(projectDir string, filePath string) (time.Time, error) {
	gitDir, err := getGitDir(projectDir)
	if err != nil {
		return time.Time{}, err
	}

	filePath, err = filepath.Rel(gitDir, filePath)
	if err != nil {
		return time.Time{}, err
	}

	repo, err := git.PlainOpen(gitDir)
	if err != nil {
		return time.Time{}, err
	}
	cIter, _ := repo.Log(&git.LogOptions{FileName: &filePath, Order: git.LogOrderCommitterTime})
	if err != nil {
		return time.Time{}, err
	}
	commit, err := cIter.Next()
	if err != nil {
		// fmt.Println(err) prints just `EOF`
		return time.Time{}, err
	}
	return commit.Author.When, nil
}
