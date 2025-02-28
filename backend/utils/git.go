package utils

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

// GitClone clones a git repository to the given parent directory.
// Returns the equivalent git command string and any error encountered.
func GitClone(url, localDir, branch string) (cmd string, err error) {
	parts := strings.Split(url, "/")
	repoName := strings.TrimSuffix(parts[len(parts)-1], ".git")
	repoPath := filepath.Join(localDir, repoName)
	cmd = "git clone --recursive " + url + " " + repoPath

	_, err = git.PlainClone(repoPath, false, &git.CloneOptions{
		URL:               url,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})
	if err != nil {
		return
	}

	if branch != "" {
		cmd += " && git checkout " + branch
		_, err = GitCheckout(repoPath, branch)
		if err != nil {
			return
		}
	}

	return
}

// GitCheckout checks out the specified branch in the repository.
// If branch is empty, it attempts to determine the default branch.
// Returns the equivalent git command string and any error encountered.
func GitCheckout(repoPath, branch string) (cmd string, err error) {
	cmd = "git checkout " + branch
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return
	}

	// Get default branch if branch is empty
	if branch == "" {
		remote, err := repo.Remote("origin")
		if err != nil {
			return cmd, fmt.Errorf("failed to get remote: %v", err)
		}

		refs, err := remote.List(&git.ListOptions{})
		if err != nil {
			return cmd, fmt.Errorf("failed to list remote refs: %v", err)
		}

		// Find the HEAD reference to determine the default branch
		for _, ref := range refs {
			if ref.Name() == plumbing.HEAD {
				// Extract branch name, e.g. refs/heads/main → main
				branch = strings.TrimPrefix(ref.Target().String(), "refs/heads/")
				break
			}
		}

		if branch == "" {
			return cmd, fmt.Errorf("could not determine default branch")
		}
		fmt.Printf("Using default branch: %s\n", branch)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return
	}

	branchRefName := plumbing.NewBranchReferenceName(branch)
	branchCoOpts := git.CheckoutOptions{
		Branch: plumbing.ReferenceName(branchRefName),
		Force:  true,
	}
	if err = worktree.Checkout(&branchCoOpts); err != nil {
		// First time checkout, creat branch and set tracking information
		mirrorRemoteBranchRefSpec := fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch)
		err = gitFetchOrigin(repo, mirrorRemoteBranchRefSpec)
		if err != nil {
			return
		}
		err = worktree.Checkout(&branchCoOpts)
		if err != nil {
			return
		}

		cfg, err := repo.Config()
		if err != nil {
			return cmd, fmt.Errorf("failed to get repo config: %v", err)
		}
		cfg.Branches[branch] = &config.Branch{
			Name:   branch,
			Remote: "origin",
			Merge:  branchRefName, // refs/heads/<branch>
		}
		err = repo.SetConfig(cfg)
		if err != nil {
			return cmd, fmt.Errorf("failed to set tracking branch: %v", err)
		}
	}

	return
}

// gitFetchOrigin fetches from the origin remote with optional refspec.
// This is a helper function used primarily by GitCheckout.
func gitFetchOrigin(repo *git.Repository, refSpecStr string) (err error) {
	remote, err := repo.Remote("origin")
	if err != nil {
		return
	}

	var refSpecs []config.RefSpec
	if refSpecStr != "" {
		refSpecs = []config.RefSpec{config.RefSpec(refSpecStr)}
	}

	if err = remote.Fetch(&git.FetchOptions{
		RefSpecs: refSpecs,
	}); err != nil {
		if err == git.NoErrAlreadyUpToDate {
			Logger.Info("refs already up to date")
		} else {
			return fmt.Errorf("fetch origin failed: %v", err)
		}
	}

	return nil
}

// GitPull performs a git pull on the specified repository.
// Returns the equivalent git command string and any error encountered.
func GitPull(repoPath string) (cmd string, err error) {
	cmd = "git pull origin"
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return
	}

	err = worktree.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return
	}

	return cmd, nil
}
