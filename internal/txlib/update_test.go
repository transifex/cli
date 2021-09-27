package txlib

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/transifex/cli/pkg/assert"
)

func TestUpdateCommandVersionLessThanProduction(t *testing.T) {
	arguments := UpdateCommandArguments{
		Version: "0.0.1",
		Check:   true,
	}
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	UpdateCommand(arguments)
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	result := string(out)
	assert.True(t, strings.Contains(
		result, "There is a new latest release for you "))
}

func TestUpdateCommandCheckGreaterThanProduction(t *testing.T) {
	arguments := UpdateCommandArguments{
		Version: "100.0.0",
		Check:   true,
	}
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	UpdateCommand(arguments)
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	result := string(out)
	assert.True(t, strings.Contains(
		result, "Congratulations, you are up to date"))
}

func TestUpdateCommandCheckEQtoProduction(t *testing.T) {
	latest, _, err := selfupdate.DetectLatest("transifex/cli")
	if err != nil {
		t.Error(err)
	}
	arguments := UpdateCommandArguments{
		Version: latest.Version.String(),
		Check:   true,
	}
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	UpdateCommand(arguments)
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	result := string(out)
	assert.True(t, strings.Contains(
		result, "Congratulations, you are up to date"))
}

func TestUpdateCommandCheckLessThanProduction(t *testing.T) {
	arguments := UpdateCommandArguments{
		Version: "0.0.1",
	}
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	UpdateCommand(arguments)
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	result := string(out)
	assert.True(t, strings.Contains(
		result, "There is a new latest release"))

	// There was a prompt that proceeded with no
	assert.True(t, strings.Contains(
		result, "Update Cancelled"))
}

func TestUpdateCommandGreaterThanProduction(t *testing.T) {
	arguments := UpdateCommandArguments{
		Version: "100.0.0",
	}
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	UpdateCommand(arguments)
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	result := string(out)
	assert.True(t, strings.Contains(
		result, "Congratulations, you are up to date"))
}

func TestUpdateCommandEQtoProduction(t *testing.T) {
	latest, _, err := selfupdate.DetectLatest("transifex/cli")
	if err != nil {
		t.Error(err)
	}
	arguments := UpdateCommandArguments{
		Version: latest.Version.String(),
	}
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	UpdateCommand(arguments)
	w.Close()
	out, _ := ioutil.ReadAll(r)
	os.Stdout = rescueStdout

	result := string(out)
	assert.True(t, strings.Contains(
		result, "Congratulations, you are up to date"))
}
