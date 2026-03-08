package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mudler/skillserver/pkg/git"
	"github.com/mudler/skillserver/pkg/persistence"
)

func newStoredGitCredentialProvider(
	gitCredentialCfg GitCredentialRuntimeConfig,
	persistenceRuntime *catalogPersistenceRuntime,
) (git.StoredGitCredentialProvider, *persistence.GitRepoCredentialRepository, error) {
	if !gitCredentialCfg.EnableStoredCredentials {
		return nil, nil, nil
	}
	if persistenceRuntime == nil || persistenceRuntime.db == nil {
		return nil, nil, fmt.Errorf("persistence runtime is required for stored git credentials")
	}
	if strings.TrimSpace(gitCredentialCfg.MasterKey) == "" {
		return nil, nil, fmt.Errorf("git credential master key is required for stored git credentials")
	}

	cipher, err := persistence.NewGitRepoCredentialCipher(
		gitCredentialCfg.MasterKey,
		persistence.GitRepoCredentialCipherOptions{},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("initialize git credential cipher: %w", err)
	}
	repo, err := persistence.NewGitRepoCredentialRepository(persistenceRuntime.db, cipher)
	if err != nil {
		return nil, nil, fmt.Errorf("initialize git credential repository: %w", err)
	}

	return &storedGitCredentialProvider{repository: repo}, repo, nil
}

type storedGitCredentialProvider struct {
	repository *persistence.GitRepoCredentialRepository
}

func (p *storedGitCredentialProvider) Resolve(repo git.GitRepoConfig) (git.ResolvedGitAuthCredentials, error) {
	if p == nil || p.repository == nil {
		return git.ResolvedGitAuthCredentials{}, git.NewRedactedGitAuthErrorf("stored credential provider is not configured")
	}

	authSource := strings.TrimSpace(strings.ToLower(repo.Auth.Source))
	if authSource != git.GitRepoAuthSourceStored {
		return git.ResolvedGitAuthCredentials{}, git.NewRedactedGitAuthErrorf("stored credential provider requires auth source %q", git.GitRepoAuthSourceStored)
	}

	referenceID := strings.TrimSpace(repo.Auth.ReferenceID)
	if referenceID == "" {
		referenceID = strings.TrimSpace(repo.ID)
	}
	if referenceID == "" {
		return git.ResolvedGitAuthCredentials{}, git.NewRedactedGitAuthErrorf("stored credential reference id is required")
	}

	payload, err := p.repository.GetCredential(context.Background(), referenceID)
	if err != nil {
		if errors.Is(err, persistence.ErrGitRepoCredentialNotFound) {
			return git.ResolvedGitAuthCredentials{}, git.NewRedactedGitAuthErrorf("stored credentials not found for repository id %q", referenceID)
		}
		return git.ResolvedGitAuthCredentials{}, git.NewRedactedGitAuthErrorf(
			"failed to load stored credentials for repository id %q: %v",
			referenceID,
			err,
		)
	}

	return mapStoredCredentialPayload(repo, payload)
}

func mapStoredCredentialPayload(
	repo git.GitRepoConfig,
	payload persistence.GitRepoCredentialSecretPayload,
) (git.ResolvedGitAuthCredentials, error) {
	mode := strings.TrimSpace(strings.ToLower(repo.Auth.Mode))
	if mode == "" || mode == git.GitRepoAuthModeNone {
		return git.ResolvedGitAuthCredentials{}, git.NewRedactedGitAuthErrorf("stored credential auth mode is required")
	}

	resolved := git.ResolvedGitAuthCredentials{
		Mode:   mode,
		Source: git.GitRepoAuthSourceStored,
	}

	switch mode {
	case git.GitRepoAuthModeHTTPSToken:
		if payload.Type != persistence.GitRepoCredentialSecretTypeHTTPSToken {
			return git.ResolvedGitAuthCredentials{}, git.NewRedactedGitAuthErrorf("stored credential payload type mismatch for mode %q", mode)
		}
		resolved.Username = strings.TrimSpace(payload.Username)
		resolved.Token = payload.Token
	case git.GitRepoAuthModeHTTPSBasic:
		if payload.Type != persistence.GitRepoCredentialSecretTypeHTTPSBasic {
			return git.ResolvedGitAuthCredentials{}, git.NewRedactedGitAuthErrorf("stored credential payload type mismatch for mode %q", mode)
		}
		resolved.Username = strings.TrimSpace(payload.Username)
		resolved.Password = payload.Password
	case git.GitRepoAuthModeSSHKey:
		if payload.Type != persistence.GitRepoCredentialSecretTypeSSHKey {
			return git.ResolvedGitAuthCredentials{}, git.NewRedactedGitAuthErrorf("stored credential payload type mismatch for mode %q", mode)
		}
		resolved.PrivateKey = payload.PrivateKey
		resolved.Passphrase = payload.Passphrase
		resolved.KnownHosts = payload.KnownHosts
	default:
		return git.ResolvedGitAuthCredentials{}, git.NewRedactedGitAuthErrorf("unsupported stored credential auth mode %q", mode)
	}

	return resolved, nil
}
