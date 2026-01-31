package core

import (
	"log/slog"
	"strings"

	"github.com/dominikospritis/git-trim/internal/git"
)

// ResolveBases resolves the base branches to use for comparison
func ResolveBases(repo *git.Repository, explicitBases []string, logger *slog.Logger) ([]string, error) {
	if len(explicitBases) > 0 {
		return validateBases(repo, explicitBases, logger)
	}

	return autoDetectBases(repo, logger)
}

func validateBases(repo *git.Repository, bases []string, logger *slog.Logger) ([]string, error) {
	var validBases []string

	for _, base := range bases {
		refname := base
		if !strings.HasPrefix(base, "refs/") {
			refname = "refs/heads/" + base
		}

		if repo.RefExists(refname) {
			validBases = append(validBases, base)
		} else {
			logger.Warn("base branch not found", "branch", base)
		}
	}

	if len(validBases) == 0 {
		logger.Warn("no valid base branches found, falling back to auto-detection")
		return autoDetectBases(repo, logger)
	}

	return validBases, nil
}

func autoDetectBases(repo *git.Repository, logger *slog.Logger) ([]string, error) {
	remotes, err := repo.Remotes()
	if err != nil {
		return nil, err
	}

	baseBranches := make(map[string]bool)

	for _, remote := range remotes {
		defaultBranch, err := repo.GetRemoteDefaultBranch(remote)
		if err != nil {
			logger.Debug("could not get default branch for remote", "remote", remote, "error", err)
			continue
		}

		trackingBranches, err := repo.TrackingBranches()
		if err != nil {
			continue
		}

		for _, tb := range trackingBranches {
			upstreamName := tb.Upstream.BranchName()
			if upstreamName == defaultBranch && tb.Upstream.RemoteName() == remote {
				baseBranches[tb.Local.Name()] = true
				logger.Debug("found base branch", "branch", tb.Local.Name(), "tracking", tb.Upstream.Name())
			}
		}

		remoteRef := "refs/remotes/" + remote + "/" + defaultBranch
		if repo.RefExists(remoteRef) {
			baseBranches[remote+"/"+defaultBranch] = true
		}
	}

	var bases []string
	for base := range baseBranches {
		bases = append(bases, base)
	}

	if len(bases) == 0 {
		logger.Debug("no remote HEAD found, trying common branch names")
		commonBases := []string{"main", "master", "develop", "development"}
		for _, name := range commonBases {
			if repo.RefExists("refs/heads/" + name) {
				bases = append(bases, name)
				break
			}
		}
	}

	if len(bases) == 0 {
		logger.Warn("could not auto-detect any base branches")
	} else {
		logger.Info("auto-detected base branches", "bases", bases)
	}

	return bases, nil
}
