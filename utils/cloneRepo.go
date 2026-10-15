package utils

import (
	"Builder/spinner"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneRepo grabs url and clones the repo
func CloneRepo(to string) {
	// Get absolute path if a relative path was given
	toPath, _ := filepath.Abs(to)

	repo := GetRepoURL()

	// Stat path, if it doesn't exist, create it
	_, err := os.Stat(toPath)
	if err != nil {
		errDir := os.MkdirAll(toPath, 0755)
		if errDir != nil {
			spinner.LogMessage("Could not create new repo directory: "+errDir.Error(), "fatal")
		}

		gitClone(repo, toPath, "")

		// Get branch name
		os.Setenv("REPO_BRANCH_NAME", GetBranchName(toPath))
	} else {
		bFlagExists, branchExists, branchName := bFlagAndBranchExists()

		if bFlagExists {
			if branchExists {
				gitClone(repo, toPath, branchName)

				// Get branch name
				os.Setenv("REPO_BRANCH_NAME", branchName)
			} else {
				spinner.LogMessage("Branch does not exists", "fatal")
			}
		} else if branchExists { // Repo branch given in builder.yaml not by -b flag
			gitClone(repo, toPath, branchName)

			// Get branch name
			os.Setenv("REPO_BRANCH_NAME", branchName)
		} else {
			gitClone(repo, toPath, "")

			// Get branch name
			os.Setenv("REPO_BRANCH_NAME", GetBranchName(toPath))
		}
	}
}

// gitClone clones repo into dest (optionally a single branch) and exits with
// git's own error output if the clone fails.
func gitClone(repo, dest, branch string) {
	args := []string{"clone"}
	if branch != "" {
		args = append(args, "-b", branch, "--single-branch")
	}
	args = append(args, "--", repo, dest)
	cmd := exec.Command("git", args...)
	spinner.LogMessage("running command: "+cmd.String(), "info")
	if out, err := cmd.CombinedOutput(); err != nil {
		spinner.LogMessage("git clone failed: "+err.Error()+": "+strings.TrimSpace(string(out)), "fatal")
	}
}

func bFlagAndBranchExists() (bool, bool, string) {
	var bFlagExists, branchExists = false, false
	args := os.Args[1:]

	branchName := os.Getenv("REPO_BRANCH")

	for i, v := range args {
		if v == "-b" || v == "--branch" {
			if len(args) <= i+1 {
				spinner.LogMessage("No Branch Name Provided", "fatal")

			} else {
				branchName = args[i+1]
				bFlagExists = true
			}
		}
	}

	if branchName != "" {
		branchExists = true
	}

	return bFlagExists, branchExists, branchName
}

// GetBranchName returns the checked-out branch in path, or "" if it can't be read.
func GetBranchName(path string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		spinner.LogMessage("could not read branch name: "+err.Error(), "warn")
		return ""
	}
	return strings.TrimSpace(string(out))
}
