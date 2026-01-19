package apikey_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/api/internal/service/apikey"
	"github.com/team-attention/cops/api/internal/service/apikey/outbound/repository/mock"
	"github.com/team-attention/cops/shared/domain"
)

var _ = Describe("APIKey Service", func() {
	var (
		logger *slog.Logger
		repo   *mock.APIKeyRepository
		svc    *apikey.Service
		ctx    context.Context
	)

	BeforeEach(func() {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
		repo = &mock.APIKeyRepository{}
		svc = apikey.NewService(logger, repo)
		ctx = context.Background()
	})

	Describe("IssueAPIKey", func() {
		Context("with valid params", func() {
			BeforeEach(func() {
				repo.CreateFunc = func(ctx context.Context, apiKey *domain.APIKey) (*domain.APIKey, error) {
					apiKey.ID = "key-123"
					return apiKey, nil
				}
			})

			It("returns plain-text key only once", func() {
				params := apikey.IssueAPIKeyParams{
					UserID: "user-123",
					Name:   "Test Key",
				}
				result, err := svc.IssueAPIKey(ctx, params)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.APIKey).To(HavePrefix("cops_"))
				Expect(len(result.APIKey)).To(Equal(37)) // cops_ (5) + 32 random chars
			})

			It("stores hashed key in repository", func() {
				var storedKey *domain.APIKey
				repo.CreateFunc = func(ctx context.Context, apiKey *domain.APIKey) (*domain.APIKey, error) {
					storedKey = apiKey
					apiKey.ID = "key-123"
					return apiKey, nil
				}

				params := apikey.IssueAPIKeyParams{
					UserID: "user-123",
					Name:   "Test Key",
				}
				result, err := svc.IssueAPIKey(ctx, params)
				Expect(err).NotTo(HaveOccurred())

				// The stored key should have a hash, not the plain key
				Expect(storedKey.KeyHash).NotTo(BeEmpty())
				Expect(storedKey.KeyHash).NotTo(Equal(result.APIKey))
				Expect(len(storedKey.KeyHash)).To(Equal(64)) // SHA-256 hex
			})
		})

		Context("with expiration", func() {
			BeforeEach(func() {
				repo.CreateFunc = func(ctx context.Context, apiKey *domain.APIKey) (*domain.APIKey, error) {
					apiKey.ID = "key-123"
					return apiKey, nil
				}
			})

			It("sets expiresAt correctly", func() {
				params := apikey.IssueAPIKeyParams{
					UserID:        "user-123",
					Name:          "Test Key",
					ExpiresInDays: 30,
				}
				result, err := svc.IssueAPIKey(ctx, params)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.KeyInfo.ExpiresAt).NotTo(BeNil())

				expectedExpiry := time.Now().Add(30 * 24 * time.Hour)
				Expect(result.KeyInfo.ExpiresAt.Unix()).To(BeNumerically("~", expectedExpiry.Unix(), 60))
			})
		})
	})

	Describe("ValidateAPIKey", func() {
		Context("with active key", func() {
			var testKey string

			BeforeEach(func() {
				testKey = "cops_abc12345678901234567890123456"
				repo.GetByHashFunc = func(ctx context.Context, keyHash string) (*domain.APIKey, error) {
					return &domain.APIKey{
						ID:        "key-123",
						UserID:    "user-123",
						Name:      "Test Key",
						KeyHash:   keyHash,
						CreatedAt: time.Now(),
					}, nil
				}
				repo.UpdateLastUsedAtFunc = func(ctx context.Context, keyID string) error {
					return nil
				}
			})

			It("returns valid=true with userID", func() {
				result, err := svc.ValidateAPIKey(ctx, testKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Valid).To(BeTrue())
				Expect(result.UserID).To(Equal("user-123"))
				Expect(result.ErrorMessage).To(BeEmpty())
			})
		})

		Context("with revoked key", func() {
			BeforeEach(func() {
				revokedAt := time.Now()
				repo.GetByHashFunc = func(ctx context.Context, keyHash string) (*domain.APIKey, error) {
					return &domain.APIKey{
						ID:        "key-123",
						UserID:    "user-123",
						Name:      "Test Key",
						KeyHash:   keyHash,
						CreatedAt: time.Now(),
						RevokedAt: &revokedAt,
					}, nil
				}
			})

			It("returns valid=false with error message", func() {
				result, err := svc.ValidateAPIKey(ctx, "cops_abc12345678901234567890123456")
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Valid).To(BeFalse())
				Expect(result.ErrorMessage).To(ContainSubstring("revoked"))
			})
		})

		Context("with expired key", func() {
			BeforeEach(func() {
				expiredAt := time.Now().Add(-24 * time.Hour)
				repo.GetByHashFunc = func(ctx context.Context, keyHash string) (*domain.APIKey, error) {
					return &domain.APIKey{
						ID:        "key-123",
						UserID:    "user-123",
						Name:      "Test Key",
						KeyHash:   keyHash,
						CreatedAt: time.Now().Add(-48 * time.Hour),
						ExpiresAt: &expiredAt,
					}, nil
				}
			})

			It("returns valid=false with error message", func() {
				result, err := svc.ValidateAPIKey(ctx, "cops_abc12345678901234567890123456")
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Valid).To(BeFalse())
				Expect(result.ErrorMessage).To(ContainSubstring("expired"))
			})
		})

		Context("with non-existent key", func() {
			BeforeEach(func() {
				repo.GetByHashFunc = func(ctx context.Context, keyHash string) (*domain.APIKey, error) {
					return nil, nil
				}
			})

			It("returns valid=false with error message", func() {
				result, err := svc.ValidateAPIKey(ctx, "cops_nonexistent123456789012345")
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Valid).To(BeFalse())
				Expect(result.ErrorMessage).To(ContainSubstring("invalid"))
			})
		})
	})

	Describe("RevokeAPIKey", func() {
		Context("with valid key owned by user", func() {
			BeforeEach(func() {
				repo.GetByIDFunc = func(ctx context.Context, keyID string) (*domain.APIKey, error) {
					return &domain.APIKey{
						ID:     "key-123",
						UserID: "user-123",
						Name:   "Test Key",
					}, nil
				}
				repo.RevokeFunc = func(ctx context.Context, keyID string) error {
					return nil
				}
			})

			It("revokes the key successfully", func() {
				params := apikey.RevokeAPIKeyParams{
					UserID: "user-123",
					KeyID:  "key-123",
				}
				err := svc.RevokeAPIKey(ctx, params)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("with key owned by different user", func() {
			BeforeEach(func() {
				repo.GetByIDFunc = func(ctx context.Context, keyID string) (*domain.APIKey, error) {
					return &domain.APIKey{
						ID:     "key-123",
						UserID: "other-user",
						Name:   "Test Key",
					}, nil
				}
			})

			It("returns error (key not found)", func() {
				params := apikey.RevokeAPIKeyParams{
					UserID: "user-123",
					KeyID:  "key-123",
				}
				err := svc.RevokeAPIKey(ctx, params)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})

		Context("with non-existent key", func() {
			BeforeEach(func() {
				repo.GetByIDFunc = func(ctx context.Context, keyID string) (*domain.APIKey, error) {
					return nil, nil
				}
			})

			It("returns error", func() {
				params := apikey.RevokeAPIKeyParams{
					UserID: "user-123",
					KeyID:  "nonexistent",
				}
				err := svc.RevokeAPIKey(ctx, params)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ListAPIKeys", func() {
		Context("with keys for user", func() {
			BeforeEach(func() {
				repo.ListByUserFunc = func(ctx context.Context, userID string, includeRevoked bool) ([]*domain.APIKey, error) {
					return []*domain.APIKey{
						{ID: "key-1", UserID: "user-123", Name: "Key 1"},
						{ID: "key-2", UserID: "user-123", Name: "Key 2"},
					}, nil
				}
			})

			It("returns list of keys", func() {
				params := apikey.ListAPIKeysParams{
					UserID:         "user-123",
					IncludeRevoked: false,
				}
				keys, err := svc.ListAPIKeys(ctx, params)
				Expect(err).NotTo(HaveOccurred())
				Expect(keys).To(HaveLen(2))
			})
		})

		Context("with repository error", func() {
			BeforeEach(func() {
				repo.ListByUserFunc = func(ctx context.Context, userID string, includeRevoked bool) ([]*domain.APIKey, error) {
					return nil, errors.New("database error")
				}
			})

			It("returns error", func() {
				params := apikey.ListAPIKeysParams{
					UserID:         "user-123",
					IncludeRevoked: false,
				}
				_, err := svc.ListAPIKeys(ctx, params)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
