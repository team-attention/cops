package rbac_test

import (
	"context"
	"errors"
	"log/slog"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/team-attention/cops/api/internal/service/core/rbac"
	"github.com/team-attention/cops/api/internal/service/core/rbac/outbound/repository/mock"
)

var _ = Describe("RBAC Service", func() {
	var (
		logger     *slog.Logger
		memberRepo *mock.OrganizationMemberRepository
		svc        *rbac.Service
		ctx        context.Context
	)

	BeforeEach(func() {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
		memberRepo = &mock.OrganizationMemberRepository{}
		svc = rbac.NewService(logger, memberRepo)
		ctx = context.Background()
	})

	Describe("CanAccessOrganization", func() {
		Context("when user is member of organization", func() {
			BeforeEach(func() {
				memberRepo.IsMemberFunc = func(ctx context.Context, userID, organizationID string) (bool, error) {
					return true, nil
				}
			})

			It("should return true, nil", func() {
				canAccess, err := svc.CanAccessOrganization(ctx, "user-123", "org-123")
				Expect(err).NotTo(HaveOccurred())
				Expect(canAccess).To(BeTrue())
			})
		})

		Context("when user is not member of organization", func() {
			BeforeEach(func() {
				memberRepo.IsMemberFunc = func(ctx context.Context, userID, organizationID string) (bool, error) {
					return false, nil
				}
			})

			It("should return false, nil", func() {
				canAccess, err := svc.CanAccessOrganization(ctx, "user-123", "org-123")
				Expect(err).NotTo(HaveOccurred())
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when userID is empty", func() {
			It("should return false, error", func() {
				canAccess, err := svc.CanAccessOrganization(ctx, "", "org-123")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("userID"))
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when organizationID is empty", func() {
			It("should return false, error", func() {
				canAccess, err := svc.CanAccessOrganization(ctx, "user-123", "")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("organizationID"))
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when membership query fails", func() {
			BeforeEach(func() {
				memberRepo.IsMemberFunc = func(ctx context.Context, userID, organizationID string) (bool, error) {
					return false, errors.New("database error")
				}
			})

			It("should return false, error", func() {
				canAccess, err := svc.CanAccessOrganization(ctx, "user-123", "org-123")
				Expect(err).To(HaveOccurred())
				Expect(canAccess).To(BeFalse())
			})
		})
	})
})
