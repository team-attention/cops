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
	"github.com/team-attention/cops/shared/domain"
)

var _ = Describe("RBAC Service", func() {
	var (
		logger      *slog.Logger
		projectRepo *mock.ProjectRepository
		memberRepo  *mock.OrganizationMemberRepository
		svc         *rbac.Service
		ctx         context.Context
	)

	BeforeEach(func() {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
		projectRepo = &mock.ProjectRepository{}
		memberRepo = &mock.OrganizationMemberRepository{}
		svc = rbac.NewService(logger, projectRepo, memberRepo)
		ctx = context.Background()
	})

	Describe("CanAccess", func() {
		Context("when user is member of project's organization", func() {
			BeforeEach(func() {
				projectRepo.GetByIDFunc = func(ctx context.Context, projectID string) (*domain.Project, error) {
					return &domain.Project{
						ProjectAbstract: domain.ProjectAbstract{ID: domain.ID(projectID)},
						OrganizationID:  domain.ID("org-123"),
					}, nil
				}
				memberRepo.IsMemberFunc = func(ctx context.Context, userID, organizationID string) (bool, error) {
					return true, nil
				}
			})

			It("should return true, nil", func() {
				canAccess, err := svc.CanAccess(ctx, "user-123", "project-123")
				Expect(err).NotTo(HaveOccurred())
				Expect(canAccess).To(BeTrue())
			})
		})

		Context("when user is not member of project's organization", func() {
			BeforeEach(func() {
				projectRepo.GetByIDFunc = func(ctx context.Context, projectID string) (*domain.Project, error) {
					return &domain.Project{
						ProjectAbstract: domain.ProjectAbstract{ID: domain.ID(projectID)},
						OrganizationID:  domain.ID("org-123"),
					}, nil
				}
				memberRepo.IsMemberFunc = func(ctx context.Context, userID, organizationID string) (bool, error) {
					return false, nil
				}
			})

			It("should return false, nil", func() {
				canAccess, err := svc.CanAccess(ctx, "user-123", "project-123")
				Expect(err).NotTo(HaveOccurred())
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when project not found", func() {
			BeforeEach(func() {
				projectRepo.GetByIDFunc = func(ctx context.Context, projectID string) (*domain.Project, error) {
					return nil, nil
				}
			})

			It("should return false, nil", func() {
				canAccess, err := svc.CanAccess(ctx, "user-123", "project-123")
				Expect(err).NotTo(HaveOccurred())
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when userID is empty", func() {
			It("should return false, error", func() {
				canAccess, err := svc.CanAccess(ctx, "", "project-123")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("userID"))
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when projectID is empty", func() {
			It("should return false, error", func() {
				canAccess, err := svc.CanAccess(ctx, "user-123", "")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("projectID"))
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when project query fails", func() {
			BeforeEach(func() {
				projectRepo.GetByIDFunc = func(ctx context.Context, projectID string) (*domain.Project, error) {
					return nil, errors.New("database error")
				}
			})

			It("should return false, error", func() {
				canAccess, err := svc.CanAccess(ctx, "user-123", "project-123")
				Expect(err).To(HaveOccurred())
				Expect(canAccess).To(BeFalse())
			})
		})

		Context("when membership query fails", func() {
			BeforeEach(func() {
				projectRepo.GetByIDFunc = func(ctx context.Context, projectID string) (*domain.Project, error) {
					return &domain.Project{
						ProjectAbstract: domain.ProjectAbstract{ID: domain.ID(projectID)},
						OrganizationID:  domain.ID("org-123"),
					}, nil
				}
				memberRepo.IsMemberFunc = func(ctx context.Context, userID, organizationID string) (bool, error) {
					return false, errors.New("database error")
				}
			})

			It("should return false, error", func() {
				canAccess, err := svc.CanAccess(ctx, "user-123", "project-123")
				Expect(err).To(HaveOccurred())
				Expect(canAccess).To(BeFalse())
			})
		})
	})
})
