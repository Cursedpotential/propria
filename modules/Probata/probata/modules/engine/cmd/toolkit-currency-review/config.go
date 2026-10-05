// Byline: Codex · GPT-6.1 · 2026-10-04.
package main

import (
	"errors"
	"os"
	"strings"

	"github.com/Cursedpotential/probata/engine/acquisition"
	"github.com/Cursedpotential/probata/engine/derive/smsthreads"
	"github.com/Cursedpotential/probata/engine/extraction/libraryvalidation"
)

const EnvApprovalKeyFile = "TOOLKIT_CURRENCY_REVIEW_APPROVAL_KEY_FILE"
const EnvCurrencyUserFile = "TOOLKIT_CURRENCY_DB_USER_FILE"
const EnvCurrencyPasswordFile = "TOOLKIT_CURRENCY_DB_PASSWORD_FILE"
const EnvCurrencyAccess = "TOOLKIT_CURRENCY_DB_ACCESS"

// fromEnv composes the existing store/service and a separate currency record-access writer from operator-only mounted files.
// Inputs: existing validation/NIM/B2/runtime settings plus currency username/password and independent approval key files.
// Outputs: protected dependencies; effects: local configuration/preflight only. No approval key or DB login is returned to stdout.
func fromEnv() (dependencies, error) {
	if strings.TrimSpace(os.Getenv(EnvCurrencyAccess)) != libraryvalidation.CurrencyAccess {
		return dependencies{}, errors.New("currency record access required")
	}
	cfg, err := acquisition.LoadObjectStorageConfigFile(os.Getenv(libraryvalidation.EnvB2ConfigFile))
	if err != nil {
		return dependencies{}, err
	}
	if err := libraryvalidation.ValidateB2StorageEndpoint(cfg.Endpoint); err != nil {
		return dependencies{}, err
	}
	client, err := acquisition.NewS3Client(cfg)
	if err != nil {
		return dependencies{}, err
	}
	service, err := libraryvalidation.NewScopedServiceFromEnv(smsthreads.S3Store{Client: client})
	if err != nil {
		return dependencies{}, err
	}
	if len(service.Verifier.CurrencyKey) < 32 {
		return dependencies{}, errors.New("currency verification key required")
	}
	user, err := libraryvalidation.ReadCredentialFile(os.Getenv(EnvCurrencyUserFile), "TOOLKIT_CURRENCY_DB_USER")
	if err != nil {
		return dependencies{}, err
	}
	password, err := libraryvalidation.ReadCredentialFile(os.Getenv(EnvCurrencyPasswordFile), "TOOLKIT_CURRENCY_DB_PASSWORD")
	if err != nil {
		return dependencies{}, err
	}
	approvalKey, err := libraryvalidation.ReadCredentialFile(os.Getenv(EnvApprovalKeyFile), "TOOLKIT_CURRENCY_REVIEW_APPROVAL_KEY")
	if err != nil || len(approvalKey) < 32 {
		return dependencies{}, errors.New("protected approval key missing")
	}
	repo, ok := service.Repository.(libraryvalidation.HTTPRepository)
	if !ok {
		return dependencies{}, errors.New("repository contract mismatch")
	}
	dbConfig := repo.DB.Config
	dbConfig.User, dbConfig.Password = user, password
	writer, err := libraryvalidation.NewAccessSQLClient(libraryvalidation.AccessConfig{DB: dbConfig, Access: libraryvalidation.CurrencyAccess, Principal: libraryvalidation.CurrencyPrincipal}, nil)
	if err != nil {
		return dependencies{}, err
	}
	review := libraryvalidation.CurrencyReviewService{Repository: service.Repository, Artifacts: service.Artifacts, Extractor: service.Extractor, Fetcher: service.Fetcher, DB: writer, SigningKey: service.SigningKey, ApprovalKey: []byte(approvalKey), CurrencyKey: service.Verifier.CurrencyKey}
	return dependencies{Review: review, Sign: func(a libraryvalidation.CurrencyReviewApproval) (libraryvalidation.CurrencyReviewApproval, error) {
		return libraryvalidation.SignCurrencyReviewApproval(a, []byte(approvalKey))
	}}, nil
}
