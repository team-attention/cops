package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/team-attention/cops/api/internal/platform/structure"
	"github.com/team-attention/cops/api/internal/platform/util/mongoutil"
	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
	"github.com/team-attention/cops/shared/domain/mongoschema"
	session "github.com/team-attention/cops/shared/domain/v2"
)

// Aggregation output field name constants.
// These are used as field names in $group/$addFields stages (must not contain dots)
// and for extracting results via mongoutil.Get.
const (
	// aggInputTokensField is the output field name for input token sums in aggregations.
	aggInputTokensField = "inputTokens"
	// aggOutputTokensField is the output field name for output token sums in aggregations.
	aggOutputTokensField = "outputTokens"
	// aggCacheReadTokensField is the output field name for cache read token sums in aggregations.
	aggCacheReadTokensField = "cacheReadTokens"
	// aggCacheCreationTokensField is the output field name for cache creation token sums in aggregations.
	aggCacheCreationTokensField = "cacheCreationTokens"
)

// MongoDashboardRepository implements DashboardRepositoryPort using MongoDB.
type MongoDashboardRepository struct {
	logger       *slog.Logger
	projectsColl *mongo.Collection
	sessionsColl *mongo.Collection
}

// NewMongoDashboardRepository creates a new MongoDB dashboard repository adapter.
func NewMongoDashboardRepository(l *slog.Logger, db *mongo.Database) *MongoDashboardRepository {
	return &MongoDashboardRepository{
		logger:       l.With(slog.String("name", "dashboard.repository.mongodb")),
		projectsColl: db.Collection(mongoschema.ProjectCollectionName),
		sessionsColl: db.Collection(mongoschema.SessionsCollectionName),
	}
}

// GetOverviewStats retrieves dashboard overview statistics for an organization.
func (r *MongoDashboardRepository) GetOverviewStats(ctx context.Context, organizationID string) (*repository.OverviewStats, error) {
	r.logger.Debug("GetOverviewStats called", slog.String("organizationID", organizationID))

	// Convert organizationID to ObjectID
	orgOID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	stats := &repository.OverviewStats{}

	// Get project IDs for this organization first
	projectIDs, err := r.getProjectIDsForOrganization(ctx, orgOID)
	if err != nil {
		r.logger.Error("failed to get project IDs for organization", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get project IDs: %w", err)
	}

	// Early return if no projects
	if len(projectIDs) == 0 {
		return stats, nil
	}

	// Get total usage from session records for projects in this organization
	usagePipeline := bson.A{
		bson.M{"$match": bson.M{mongoschema.SessionProjectIDField: bson.M{"$in": projectIDs}}},
		bson.M{"$group": bson.M{
			"_id":                      nil,
			"totalInputTokens":         bson.M{"$sum": "$" + mongoschema.SessionUsageInputTokensPath},
			"totalOutputTokens":        bson.M{"$sum": "$" + mongoschema.SessionUsageOutputTokensPath},
			"totalCacheCreationTokens": bson.M{"$sum": "$" + mongoschema.SessionUsageCacheCreationInputTokensPath},
			"totalCacheReadTokens":     bson.M{"$sum": "$" + mongoschema.SessionUsageCacheReadInputTokensPath},
		}},
	}

	usageCursor, err := r.sessionsColl.Aggregate(ctx, usagePipeline)
	if err != nil {
		r.logger.Error("failed to aggregate total usage", slog.Any("error", err))
		return nil, fmt.Errorf("failed to aggregate total usage: %w", err)
	}
	defer usageCursor.Close(ctx)

	if usageCursor.Next(ctx) {
		var result struct {
			TotalInputTokens         int64 `bson:"totalInputTokens"`
			TotalOutputTokens        int64 `bson:"totalOutputTokens"`
			TotalCacheCreationTokens int64 `bson:"totalCacheCreationTokens"`
			TotalCacheReadTokens     int64 `bson:"totalCacheReadTokens"`
		}
		if err := usageCursor.Decode(&result); err == nil {
			stats.TotalUsage = repository.TokenUsageSummary{
				TotalInputTokens:         result.TotalInputTokens,
				TotalOutputTokens:        result.TotalOutputTokens,
				TotalCacheCreationTokens: result.TotalCacheCreationTokens,
				TotalCacheReadTokens:     result.TotalCacheReadTokens,
			}
		}
	}

	// Get project count for this organization
	projectCount, err := r.projectsColl.CountDocuments(ctx, bson.M{mongoschema.ProjectOrganizationIDField: orgOID})
	if err != nil {
		r.logger.Error("failed to count projects", slog.Any("error", err))
		return nil, fmt.Errorf("failed to count projects: %w", err)
	}
	stats.ProjectCount = int32(projectCount)
	r.logger.Debug("overview stats counts",
		slog.Int64("projectCount", projectCount),
	)

	// Get session count (distinct session_ids) for projects in this organization
	sessionPipeline := bson.A{
		bson.M{"$match": bson.M{mongoschema.SessionProjectIDField: bson.M{"$in": projectIDs}}},
		bson.M{"$group": bson.M{"_id": "$" + mongoschema.SessionSessionIDField}},
		bson.M{"$count": "count"},
	}

	sessionCursor, err := r.sessionsColl.Aggregate(ctx, sessionPipeline)
	if err != nil {
		r.logger.Error("failed to count sessions", slog.Any("error", err))
		return nil, fmt.Errorf("failed to count sessions: %w", err)
	}
	defer sessionCursor.Close(ctx)

	if sessionCursor.Next(ctx) {
		var result struct {
			Count int32 `bson:"count"`
		}
		if err := sessionCursor.Decode(&result); err == nil {
			stats.SessionCount = result.Count
		}
	}

	// Get recent projects (top 5 by last activity) for this organization
	recentProjects, err := r.getRecentProjects(ctx, orgOID, 5)
	if err != nil {
		r.logger.Error("failed to get recent projects", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get recent projects: %w", err)
	}
	stats.RecentProjects = recentProjects

	// Get recent sessions (top 5 by started_at) for projects in this organization
	recentSessions, err := r.getRecentSessions(ctx, projectIDs, 5)
	if err != nil {
		r.logger.Error("failed to get recent sessions", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get recent sessions: %w", err)
	}
	stats.RecentSessions = recentSessions

	return stats, nil
}

// ListProjects retrieves a paginated list of projects for an organization.
func (r *MongoDashboardRepository) ListProjects(ctx context.Context, params repository.ListProjectsParams) (*repository.PaginatedProjects, error) {
	r.logger.Debug("ListProjects called",
		slog.String("organizationID", params.Query.OrganizationID),
		slog.Int("page", int(params.Page)),
		slog.Int("pageSize", int(params.PageSize)),
	)

	// Convert organizationID to ObjectID
	orgOID, err := bson.ObjectIDFromHex(params.Query.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	// Single aggregation pipeline with $facet for count and data
	pipeline := bson.A{
		// Stage 0: Match by organization
		bson.M{"$match": bson.M{mongoschema.ProjectOrganizationIDField: orgOID}},

		// Stage 1: Lookup session records with sub-pipeline for aggregation
		bson.M{"$lookup": bson.M{
			"from": mongoschema.SessionsCollectionName,
			"let":  bson.M{"projectId": "$_id"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{
					"$expr": bson.M{"$eq": bson.A{"$" + mongoschema.SessionProjectIDField, "$$projectId"}},
				}},
				bson.M{"$group": bson.M{
					"_id":                       nil,
					"sessionIds":                bson.M{"$addToSet": "$" + mongoschema.SessionSessionIDField},
					"lastActivity":              bson.M{"$max": "$" + mongoschema.SessionTimestampField},
					aggInputTokensField:         bson.M{"$sum": "$" + mongoschema.SessionUsageInputTokensPath},
					aggOutputTokensField:        bson.M{"$sum": "$" + mongoschema.SessionUsageOutputTokensPath},
					aggCacheCreationTokensField: bson.M{"$sum": "$" + mongoschema.SessionUsageCacheCreationInputTokensPath},
					aggCacheReadTokensField:     bson.M{"$sum": "$" + mongoschema.SessionUsageCacheReadInputTokensPath},
				}},
				bson.M{"$project": bson.M{
					"_id":                       0,
					"sessionCount":              bson.M{"$size": "$sessionIds"},
					"lastActivity":              1,
					aggInputTokensField:         1,
					aggOutputTokensField:        1,
					aggCacheCreationTokensField: 1,
					aggCacheReadTokensField:     1,
				}},
			},
			"as": "stats",
		}},

		// Stage 3: Unwind stats (preserveNullAndEmptyArrays for projects without sessions)
		bson.M{"$unwind": bson.M{
			"path":                       "$stats",
			"preserveNullAndEmptyArrays": true,
		}},

		// Stage 4: Project final fields with defaults
		bson.M{"$project": bson.M{
			mongoschema.ProjectIDField:        1,
			mongoschema.ProjectNameField:      1,
			mongoschema.ProjectPathField:      1,
			mongoschema.ProjectGitBranchField: 1,
			"sessionCount":                    bson.M{"$ifNull": bson.A{"$stats.sessionCount", 0}},
			"lastActivity":                    bson.M{"$ifNull": bson.A{"$stats.lastActivity", nil}},
			aggInputTokensField:               bson.M{"$ifNull": bson.A{"$stats." + aggInputTokensField, 0}},
			aggOutputTokensField:              bson.M{"$ifNull": bson.A{"$stats." + aggOutputTokensField, 0}},
			aggCacheCreationTokensField:       bson.M{"$ifNull": bson.A{"$stats." + aggCacheCreationTokensField, 0}},
			aggCacheReadTokensField:           bson.M{"$ifNull": bson.A{"$stats." + aggCacheReadTokensField, 0}},
		}},

		// Stage 5: Facet for count and paginated data
		bson.M{"$facet": bson.M{
			"metadata": bson.A{
				bson.M{"$count": "totalCount"},
			},
			"data": bson.A{
				bson.M{"$sort": bson.M{"name": 1}},
				bson.M{"$skip": params.Skip()},
				bson.M{"$limit": int64(params.PageSize)},
			},
		}},
	}

	cursor, err := r.projectsColl.Aggregate(ctx, pipeline)
	if err != nil {
		r.logger.Error("failed to aggregate projects", slog.Any("error", err))
		return nil, fmt.Errorf("failed to aggregate projects: %w", err)
	}
	defer cursor.Close(ctx)

	// Parse facet result
	var facetResult struct {
		Metadata []struct {
			TotalCount int64 `bson:"totalCount"`
		} `bson:"metadata"`
		Data []bson.M `bson:"data"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&facetResult); err != nil {
			r.logger.Error("failed to decode facet result", slog.Any("error", err))
			return nil, fmt.Errorf("failed to decode facet result: %w", err)
		}
	}

	// Extract total count
	var totalCount int64
	if len(facetResult.Metadata) > 0 {
		totalCount = facetResult.Metadata[0].TotalCount
	}
	r.logger.Debug("ListProjects results",
		slog.Int64("totalCount", totalCount),
		slog.Int("dataCount", len(facetResult.Data)),
	)

	// Convert to ProjectSummary slice
	projects := make([]repository.ProjectSummary, 0, len(facetResult.Data))
	for _, doc := range facetResult.Data {
		project := repository.ProjectSummary{
			ProjectAbstract: shareddomain.ProjectAbstract{
				ID:   shareddomain.ID(doc["_id"].(bson.ObjectID).Hex()),
				Name: mongoutil.Get[string](doc, mongoschema.ProjectNameField),
				Path: mongoutil.Get[string](doc, mongoschema.ProjectPathField),
			},
			GitBranch: mongoutil.Get[string](doc, mongoschema.ProjectGitBranchField),
			ProjectAggregation: repository.ProjectAggregation{
				SessionCount: mongoutil.Get[int32](doc, "sessionCount"),
				LastActivity: mongoutil.Get[time.Time](doc, "lastActivity"),
				Usage: repository.TokenUsageSummary{
					TotalInputTokens:         mongoutil.Get[int64](doc, aggInputTokensField),
					TotalOutputTokens:        mongoutil.Get[int64](doc, aggOutputTokensField),
					TotalCacheCreationTokens: mongoutil.Get[int64](doc, aggCacheCreationTokensField),
					TotalCacheReadTokens:     mongoutil.Get[int64](doc, aggCacheReadTokensField),
				},
			},
		}
		projects = append(projects, project)
	}

	return structure.NewPaginatedResult(projects, params.Page, params.PageSize, totalCount), nil
}

// GetProject retrieves detailed project information.
// Returns nil, error if project not found or does not belong to organization.
func (r *MongoDashboardRepository) GetProject(ctx context.Context, organizationID, projectID string) (*repository.ProjectDetail, error) {
	orgOID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	objectID, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	// Query with both conditions: project ID and organization ID
	var doc bson.M
	err = r.projectsColl.FindOne(ctx, bson.M{
		"_id":                              objectID,
		mongoschema.ProjectOrganizationIDField: orgOID,
	}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("project not found")
		}
		r.logger.Error("failed to find project", slog.String("projectID", projectID), slog.Any("error", err))
		return nil, fmt.Errorf("failed to find project: %w", err)
	}

	detail := &repository.ProjectDetail{
		Project: shareddomain.Project{
			ProjectAbstract: shareddomain.ProjectAbstract{
				ID:   shareddomain.ID(projectID),
				Name: mongoutil.Get[string](doc, mongoschema.ProjectNameField),
				Path: mongoutil.Get[string](doc, mongoschema.ProjectPathField),
			},
			IsGitProject: mongoutil.Get[bool](doc, mongoschema.ProjectIsGitProjectField),
			RegisteredAt: mongoutil.Get[time.Time](doc, mongoschema.ProjectRegisteredAtField),
		},
	}

	// Get session stats
	stats, err := r.getProjectStats(ctx, projectID)
	if err == nil {
		detail.ProjectAggregation = repository.ProjectAggregation{
			SessionCount: stats.SessionCount,
			Usage:        stats.Usage,
			LastActivity: stats.LastActivity,
		}
	}

	return detail, nil
}

// ListSessions retrieves paginated sessions for a project in an organization.
func (r *MongoDashboardRepository) ListSessions(ctx context.Context, params repository.ListSessionsParams) (*repository.PaginatedSessions, error) {
	// Convert organizationID to ObjectID
	orgOID, err := bson.ObjectIDFromHex(params.Query.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	// Get project IDs for this organization
	projectIDs, err := r.getProjectIDsForOrganization(ctx, orgOID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project IDs: %w", err)
	}

	// Build aggregation pipeline
	pipeline := bson.A{}

	// Filter by projects in this organization
	if params.Query.ProjectID != "" {
		// Specific project requested - verify it belongs to organization
		projectOID, err := bson.ObjectIDFromHex(params.Query.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("invalid project ID: %w", err)
		}
		// Check if this project is in the organization's projects
		found := false
		for _, pid := range projectIDs {
			if pid == projectOID {
				found = true
				break
			}
		}
		if !found {
			// Project not in organization, return empty result
			return structure.NewPaginatedResult([]repository.SessionSummary{}, params.Page, params.PageSize, 0), nil
		}
		pipeline = append(pipeline, bson.M{"$match": bson.M{mongoschema.SessionProjectIDField: projectOID}})
	} else {
		// All projects in organization
		if len(projectIDs) == 0 {
			return structure.NewPaginatedResult([]repository.SessionSummary{}, params.Page, params.PageSize, 0), nil
		}
		pipeline = append(pipeline, bson.M{"$match": bson.M{mongoschema.SessionProjectIDField: bson.M{"$in": projectIDs}}})
	}

	// Filter out metadata types (summary, file-history-snapshot) that don't have sessionId.
	// These types are design-specific and should not be included in session aggregation.
	pipeline = append(pipeline, bson.M{"$match": bson.M{
		mongoschema.SessionSessionIDField: bson.M{
			"$exists": true,
			"$ne":     "",
		},
	}})

	// Add group stage
	pipeline = append(pipeline, bson.M{"$group": bson.M{
		"_id":                             "$" + mongoschema.SessionSessionIDField,
		"messageCount":                    bson.M{"$sum": 1},
		"startedAt":                       bson.M{"$min": "$" + mongoschema.SessionTimestampField},
		"endedAt":                         bson.M{"$max": "$" + mongoschema.SessionTimestampField},
		mongoschema.SessionProjectIDField: bson.M{"$first": "$" + mongoschema.SessionProjectIDField},
		mongoschema.SessionGitBranchField: bson.M{"$first": "$" + mongoschema.SessionGitBranchField},
		aggInputTokensField:               bson.M{"$sum": "$" + mongoschema.SessionUsageInputTokensPath},
		aggOutputTokensField:              bson.M{"$sum": "$" + mongoschema.SessionUsageOutputTokensPath},
		aggCacheCreationTokensField:       bson.M{"$sum": "$" + mongoschema.SessionUsageCacheCreationInputTokensPath},
		aggCacheReadTokensField:           bson.M{"$sum": "$" + mongoschema.SessionUsageCacheReadInputTokensPath},
		"totalTokens": bson.M{"$sum": bson.M{"$add": bson.A{
			"$" + mongoschema.SessionUsageInputTokensPath,
			"$" + mongoschema.SessionUsageOutputTokensPath,
		}}},
	}})

	// Sort field mapping (API snake_case -> MongoDB camelCase aggregation fields)
	sortFieldMap := map[string]string{
		"started_at":    "endedAt",      // Sort by last activity for better UX
		"message_count": "messageCount",
		"usage":         "totalTokens",
	}

	// Add sort
	sortField := "endedAt" // Default: sort by last activity
	if params.Query.SortBy != "" {
		if mapped, ok := sortFieldMap[params.Query.SortBy]; ok {
			sortField = mapped
		}
	}
	sortOrder := -1
	if !params.Query.SortDesc {
		sortOrder = 1
	}
	pipeline = append(pipeline, bson.M{"$sort": bson.M{sortField: sortOrder}})

	// Get total count first
	countPipeline := append(pipeline, bson.M{"$count": "count"})
	countCursor, err := r.sessionsColl.Aggregate(ctx, countPipeline)
	if err != nil {
		r.logger.Error("failed to count sessions", slog.Any("error", err))
		return nil, fmt.Errorf("failed to count sessions: %w", err)
	}
	defer countCursor.Close(ctx)

	var totalCount int64
	if countCursor.Next(ctx) {
		var result struct {
			Count int64 `bson:"count"`
		}
		if err := countCursor.Decode(&result); err == nil {
			totalCount = result.Count
		}
	}

	// Add pagination
	pipeline = append(pipeline,
		bson.M{"$skip": params.Skip()},
		bson.M{"$limit": int64(params.PageSize)},
	)

	cursor, err := r.sessionsColl.Aggregate(ctx, pipeline)
	if err != nil {
		r.logger.Error("failed to aggregate sessions", slog.Any("error", err))
		return nil, fmt.Errorf("failed to aggregate sessions: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []repository.SessionSummary
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		session := repository.SessionSummary{
			SessionBase: repository.SessionBase{
				ID:        mongoutil.Get[string](doc, "_id"),
				ProjectID: doc[mongoschema.SessionProjectIDField].(bson.ObjectID).Hex(),
				GitBranch: mongoutil.Get[string](doc, mongoschema.SessionGitBranchField),
				StartedAt: mongoutil.Get[time.Time](doc, "startedAt"),
				EndedAt:   mongoutil.Get[time.Time](doc, "endedAt"),
			},
			MessageCount: mongoutil.Get[int32](doc, "messageCount"),
			Usage: repository.TokenUsageSummary{
				TotalInputTokens:         mongoutil.Get[int64](doc, aggInputTokensField),
				TotalOutputTokens:        mongoutil.Get[int64](doc, aggOutputTokensField),
				TotalCacheCreationTokens: mongoutil.Get[int64](doc, aggCacheCreationTokensField),
				TotalCacheReadTokens:     mongoutil.Get[int64](doc, aggCacheReadTokensField),
			},
		}

		sessions = append(sessions, session)
	}

	return structure.NewPaginatedResult(sessions, params.Page, params.PageSize, totalCount), nil
}

// GetSession retrieves detailed session information with paginated transcripts.
// Returns nil, error if session not found or its project does not belong to organization.
func (r *MongoDashboardRepository) GetSession(ctx context.Context, params repository.GetSessionParams) (*repository.PaginatedSessionDetail, error) {
	organizationID := params.Query.OrganizationID
	sessionID := params.Query.SessionID

	// Convert organizationID to ObjectID
	orgOID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	// Step 1: Get session metadata (version, usage, timestamps) using aggregation
	metaPipeline := bson.A{
		bson.M{"$match": bson.M{mongoschema.SessionSessionIDField: sessionID}},
		bson.M{"$group": bson.M{
			"_id":                            "$" + mongoschema.SessionSessionIDField,
			mongoschema.SessionProjectIDField: bson.M{"$first": "$" + mongoschema.SessionProjectIDField},
			mongoschema.SessionVersionField:   bson.M{"$first": "$" + mongoschema.SessionVersionField},
			mongoschema.SessionGitBranchField: bson.M{"$first": "$" + mongoschema.SessionGitBranchField},
			"startedAt":                      bson.M{"$min": "$" + mongoschema.SessionTimestampField},
			"endedAt":                        bson.M{"$max": "$" + mongoschema.SessionTimestampField},
			aggInputTokensField:              bson.M{"$sum": "$" + mongoschema.SessionUsageInputTokensPath},
			aggOutputTokensField:             bson.M{"$sum": "$" + mongoschema.SessionUsageOutputTokensPath},
			aggCacheCreationTokensField:      bson.M{"$sum": "$" + mongoschema.SessionUsageCacheCreationInputTokensPath},
			aggCacheReadTokensField:          bson.M{"$sum": "$" + mongoschema.SessionUsageCacheReadInputTokensPath},
		}},
	}

	metaCursor, err := r.sessionsColl.Aggregate(ctx, metaPipeline)
	if err != nil {
		r.logger.Error("failed to aggregate session metadata", slog.String("sessionID", sessionID), slog.Any("error", err))
		return nil, fmt.Errorf("failed to aggregate session metadata: %w", err)
	}
	defer metaCursor.Close(ctx)

	if !metaCursor.Next(ctx) {
		return nil, fmt.Errorf("session not found")
	}

	var metaDoc bson.M
	if err := metaCursor.Decode(&metaDoc); err != nil {
		r.logger.Error("failed to decode session metadata", slog.String("sessionID", sessionID), slog.Any("error", err))
		return nil, fmt.Errorf("failed to decode session metadata: %w", err)
	}

	projectOID, ok := metaDoc[mongoschema.SessionProjectIDField].(bson.ObjectID)
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	// Verify project belongs to organization
	projectCount, err := r.projectsColl.CountDocuments(ctx, bson.M{
		"_id":                                  projectOID,
		mongoschema.ProjectOrganizationIDField: orgOID,
	})
	if err != nil {
		r.logger.Error("failed to verify project organization", slog.Any("error", err))
		return nil, fmt.Errorf("failed to verify project: %w", err)
	}
	if projectCount == 0 {
		return nil, fmt.Errorf("session not found")
	}

	// Step 2: Get total session entry count
	totalCount, err := r.sessionsColl.CountDocuments(ctx, bson.M{mongoschema.SessionSessionIDField: sessionID})
	if err != nil {
		r.logger.Error("failed to count session entries", slog.String("sessionID", sessionID), slog.Any("error", err))
		return nil, fmt.Errorf("failed to count session entries: %w", err)
	}

	// Step 3: Get paginated session entries
	findOpts := options.Find().
		SetSort(bson.M{mongoschema.SessionTimestampField: 1}).
		SetSkip(params.Skip()).
		SetLimit(int64(params.PageSize))

	cursor, err := r.sessionsColl.Find(ctx, bson.M{mongoschema.SessionSessionIDField: sessionID}, findOpts)
	if err != nil {
		r.logger.Error("failed to find session entries", slog.String("sessionID", sessionID), slog.Any("error", err))
		return nil, fmt.Errorf("failed to find session entries: %w", err)
	}
	defer cursor.Close(ctx)

	var sessions []*session.Session
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		// Unmarshal session document based on type discriminator
		s := unmarshalSession(doc)
		if s != nil {
			sessions = append(sessions, s)
		}
	}

	// Build result
	result := &repository.PaginatedSessionDetail{
		SessionDetail: repository.SessionDetail{
			SessionBase: repository.SessionBase{
				ID:        sessionID,
				ProjectID: projectOID.Hex(),
				GitBranch: mongoutil.Get[string](metaDoc, mongoschema.SessionGitBranchField),
				StartedAt: mongoutil.Get[time.Time](metaDoc, "startedAt"),
				EndedAt:   mongoutil.Get[time.Time](metaDoc, "endedAt"),
			},
			Version: mongoutil.Get[string](metaDoc, mongoschema.SessionVersionField),
			Usage: repository.TokenUsageSummary{
				TotalInputTokens:         mongoutil.Get[int64](metaDoc, aggInputTokensField),
				TotalOutputTokens:        mongoutil.Get[int64](metaDoc, aggOutputTokensField),
				TotalCacheCreationTokens: mongoutil.Get[int64](metaDoc, aggCacheCreationTokensField),
				TotalCacheReadTokens:     mongoutil.Get[int64](metaDoc, aggCacheReadTokensField),
			},
			Sessions: sessions,
		},
		PaginationMeta: structure.NewPaginationMeta(params.Page, params.PageSize, totalCount),
	}

	return result, nil
}

// Helper methods

// getProjectIDsForOrganization returns all project IDs for an organization.
func (r *MongoDashboardRepository) getProjectIDsForOrganization(ctx context.Context, orgOID bson.ObjectID) ([]bson.ObjectID, error) {
	cursor, err := r.projectsColl.Find(ctx, bson.M{mongoschema.ProjectOrganizationIDField: orgOID}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	projectIDs := []bson.ObjectID{}
	for cursor.Next(ctx) {
		var doc struct {
			ID bson.ObjectID `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		projectIDs = append(projectIDs, doc.ID)
	}
	return projectIDs, nil
}

func (r *MongoDashboardRepository) getRecentProjects(ctx context.Context, orgOID bson.ObjectID, limit int) ([]repository.ProjectSummary, error) {
	// Get projects sorted by last activity with stats calculated via sub-pipeline
	// Using sub-pipeline pattern to avoid loading all events into memory
	pipeline := bson.A{
		// Stage 0: Match by organization
		bson.M{"$match": bson.M{mongoschema.ProjectOrganizationIDField: orgOID}},

		// Stage 1: Lookup with sub-pipeline to aggregate stats (avoids loading all events)
		bson.M{"$lookup": bson.M{
			"from": mongoschema.SessionsCollectionName,
			"let":  bson.M{"projectId": "$_id"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{
					"$expr": bson.M{"$eq": bson.A{"$" + mongoschema.SessionProjectIDField, "$$projectId"}},
				}},
				bson.M{"$group": bson.M{
					"_id":                       nil,
					"sessionIds":                bson.M{"$addToSet": "$" + mongoschema.SessionSessionIDField},
					"lastActivity":              bson.M{"$max": "$" + mongoschema.SessionTimestampField},
					aggInputTokensField:         bson.M{"$sum": "$" + mongoschema.SessionUsageInputTokensPath},
					aggOutputTokensField:        bson.M{"$sum": "$" + mongoschema.SessionUsageOutputTokensPath},
					aggCacheCreationTokensField: bson.M{"$sum": "$" + mongoschema.SessionUsageCacheCreationInputTokensPath},
					aggCacheReadTokensField:     bson.M{"$sum": "$" + mongoschema.SessionUsageCacheReadInputTokensPath},
				}},
				bson.M{"$project": bson.M{
					"_id":                       0,
					"sessionCount":              bson.M{"$size": "$sessionIds"},
					"lastActivity":              1,
					aggInputTokensField:         1,
					aggOutputTokensField:        1,
					aggCacheCreationTokensField: 1,
					aggCacheReadTokensField:     1,
				}},
			},
			"as": "stats",
		}},

		// Stage 2: Unwind stats (preserveNullAndEmptyArrays for projects without events)
		bson.M{"$unwind": bson.M{
			"path":                       "$stats",
			"preserveNullAndEmptyArrays": true,
		}},

		// Stage 3: Project final fields with defaults
		bson.M{"$project": bson.M{
			mongoschema.ProjectIDField:        1,
			mongoschema.ProjectNameField:      1,
			mongoschema.ProjectPathField:      1,
			mongoschema.ProjectGitBranchField: 1,
			"sessionCount":                    bson.M{"$ifNull": bson.A{"$stats.sessionCount", 0}},
			"lastActivity":                    bson.M{"$ifNull": bson.A{"$stats.lastActivity", nil}},
			aggInputTokensField:               bson.M{"$ifNull": bson.A{"$stats." + aggInputTokensField, 0}},
			aggOutputTokensField:              bson.M{"$ifNull": bson.A{"$stats." + aggOutputTokensField, 0}},
			aggCacheCreationTokensField:       bson.M{"$ifNull": bson.A{"$stats." + aggCacheCreationTokensField, 0}},
			aggCacheReadTokensField:           bson.M{"$ifNull": bson.A{"$stats." + aggCacheReadTokensField, 0}},
		}},

		// Stage 4: Sort by last activity (descending), nulls last
		bson.M{"$sort": bson.M{"lastActivity": -1}},

		// Stage 5: Limit results
		bson.M{"$limit": limit},
	}

	cursor, err := r.projectsColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var projects []repository.ProjectSummary
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		project := repository.ProjectSummary{
			ProjectAbstract: shareddomain.ProjectAbstract{
				ID:   shareddomain.ID(doc["_id"].(bson.ObjectID).Hex()),
				Name: mongoutil.Get[string](doc, mongoschema.ProjectNameField),
				Path: mongoutil.Get[string](doc, mongoschema.ProjectPathField),
			},
			GitBranch: mongoutil.Get[string](doc, mongoschema.ProjectGitBranchField),
			ProjectAggregation: repository.ProjectAggregation{
				SessionCount: mongoutil.Get[int32](doc, "sessionCount"),
				LastActivity: mongoutil.Get[time.Time](doc, "lastActivity"),
				Usage: repository.TokenUsageSummary{
					TotalInputTokens:         mongoutil.Get[int64](doc, aggInputTokensField),
					TotalOutputTokens:        mongoutil.Get[int64](doc, aggOutputTokensField),
					TotalCacheCreationTokens: mongoutil.Get[int64](doc, aggCacheCreationTokensField),
					TotalCacheReadTokens:     mongoutil.Get[int64](doc, aggCacheReadTokensField),
				},
			},
		}

		projects = append(projects, project)
	}

	return projects, nil
}

func (r *MongoDashboardRepository) getRecentSessions(ctx context.Context, projectIDs []bson.ObjectID, limit int) ([]repository.SessionSummary, error) {
	if len(projectIDs) == 0 {
		return []repository.SessionSummary{}, nil
	}

	pipeline := bson.A{
		// Match records for projects in the organization
		bson.M{"$match": bson.M{mongoschema.SessionProjectIDField: bson.M{"$in": projectIDs}}},
		// Filter out metadata types (summary, file-history-snapshot) that don't have sessionId.
		bson.M{"$match": bson.M{
			mongoschema.SessionSessionIDField: bson.M{
				"$exists": true,
				"$ne":     "",
			},
		}},
		bson.M{"$group": bson.M{
			"_id":                             "$" + mongoschema.SessionSessionIDField,
			"messageCount":                    bson.M{"$sum": 1},
			"startedAt":                       bson.M{"$min": "$" + mongoschema.SessionTimestampField},
			"endedAt":                         bson.M{"$max": "$" + mongoschema.SessionTimestampField},
			mongoschema.SessionProjectIDField: bson.M{"$first": "$" + mongoschema.SessionProjectIDField},
			mongoschema.SessionGitBranchField: bson.M{"$first": "$" + mongoschema.SessionGitBranchField},
			aggInputTokensField:               bson.M{"$sum": "$" + mongoschema.SessionUsageInputTokensPath},
			aggOutputTokensField:              bson.M{"$sum": "$" + mongoschema.SessionUsageOutputTokensPath},
			aggCacheCreationTokensField:       bson.M{"$sum": "$" + mongoschema.SessionUsageCacheCreationInputTokensPath},
			aggCacheReadTokensField:           bson.M{"$sum": "$" + mongoschema.SessionUsageCacheReadInputTokensPath},
		}},
		bson.M{"$sort": bson.M{"startedAt": -1}},
		bson.M{"$limit": limit},
	}

	cursor, err := r.sessionsColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var sessions []repository.SessionSummary
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		projectID := ""
		if oid, ok := doc[mongoschema.SessionProjectIDField].(bson.ObjectID); ok {
			projectID = oid.Hex()
		}

		session := repository.SessionSummary{
			SessionBase: repository.SessionBase{
				ID:        mongoutil.Get[string](doc, "_id"),
				ProjectID: projectID,
				GitBranch: mongoutil.Get[string](doc, mongoschema.SessionGitBranchField),
				StartedAt: mongoutil.Get[time.Time](doc, "startedAt"),
				EndedAt:   mongoutil.Get[time.Time](doc, "endedAt"),
			},
			MessageCount: mongoutil.Get[int32](doc, "messageCount"),
			Usage: repository.TokenUsageSummary{
				TotalInputTokens:         mongoutil.Get[int64](doc, aggInputTokensField),
				TotalOutputTokens:        mongoutil.Get[int64](doc, aggOutputTokensField),
				TotalCacheCreationTokens: mongoutil.Get[int64](doc, aggCacheCreationTokensField),
				TotalCacheReadTokens:     mongoutil.Get[int64](doc, aggCacheReadTokensField),
			},
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (r *MongoDashboardRepository) getProjectStats(ctx context.Context, projectID string) (*repository.ProjectSummary, error) {
	projectOID, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, err
	}

	pipeline := bson.A{
		bson.M{"$match": bson.M{mongoschema.SessionProjectIDField: projectOID}},
		bson.M{"$group": bson.M{
			"_id":                       nil,
			"sessionCount":              bson.M{"$addToSet": "$" + mongoschema.SessionSessionIDField},
			"lastActivity":              bson.M{"$max": "$" + mongoschema.SessionTimestampField},
			aggInputTokensField:         bson.M{"$sum": "$" + mongoschema.SessionUsageInputTokensPath},
			aggOutputTokensField:        bson.M{"$sum": "$" + mongoschema.SessionUsageOutputTokensPath},
			aggCacheCreationTokensField: bson.M{"$sum": "$" + mongoschema.SessionUsageCacheCreationInputTokensPath},
			aggCacheReadTokensField:     bson.M{"$sum": "$" + mongoschema.SessionUsageCacheReadInputTokensPath},
		}},
		bson.M{"$project": bson.M{
			"sessionCount":              bson.M{"$size": "$sessionCount"},
			"lastActivity":              1,
			aggInputTokensField:         1,
			aggOutputTokensField:        1,
			aggCacheCreationTokensField: 1,
			aggCacheReadTokensField:     1,
		}},
	}

	cursor, err := r.sessionsColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		return &repository.ProjectSummary{}, nil
	}

	var doc bson.M
	if err := cursor.Decode(&doc); err != nil {
		return nil, err
	}

	return &repository.ProjectSummary{
		ProjectAggregation: repository.ProjectAggregation{
			SessionCount: mongoutil.Get[int32](doc, "sessionCount"),
			LastActivity: mongoutil.Get[time.Time](doc, "lastActivity"),
			Usage: repository.TokenUsageSummary{
				TotalInputTokens:         mongoutil.Get[int64](doc, aggInputTokensField),
				TotalOutputTokens:        mongoutil.Get[int64](doc, aggOutputTokensField),
				TotalCacheCreationTokens: mongoutil.Get[int64](doc, aggCacheCreationTokensField),
				TotalCacheReadTokens:     mongoutil.Get[int64](doc, aggCacheReadTokensField),
			},
		},
	}, nil
}

// unmarshalSession converts a BSON document to a session.Session based on type discriminator.
func unmarshalSession(doc bson.M) *session.Session {
	// Extract type field
	typeVal, ok := doc[mongoschema.SessionTypeField]
	if !ok {
		return nil
	}

	sessionType, ok := typeVal.(string)
	if !ok {
		return nil
	}

	// Re-marshal document to BSON bytes
	docBytes, err := bson.Marshal(doc)
	if err != nil {
		return nil
	}

	// Create concrete type based on session type and unmarshal
	var data any
	switch session.SessionType(sessionType) {
	case session.SessionTypeHuman:
		var msg session.HumanMessage
		if err := bson.Unmarshal(docBytes, &msg); err != nil {
			return nil
		}
		data = &msg
	case session.SessionTypeAgent:
		var msg session.AgentMessage
		if err := bson.Unmarshal(docBytes, &msg); err != nil {
			return nil
		}
		data = &msg
	case session.SessionTypeToolExecution:
		var msg session.ToolExecution
		if err := bson.Unmarshal(docBytes, &msg); err != nil {
			return nil
		}
		data = &msg
	case session.SessionTypeSystem:
		var msg session.SystemMessage
		if err := bson.Unmarshal(docBytes, &msg); err != nil {
			return nil
		}
		data = &msg
	case session.SessionTypeProgress:
		var msg session.Progress
		if err := bson.Unmarshal(docBytes, &msg); err != nil {
			return nil
		}
		data = &msg
	default:
		return nil
	}

	return &session.Session{
		Type: session.SessionType(sessionType),
		Data: data,
	}
}

// Compile-time interface verification.
var _ repository.DashboardRepositoryPort = (*MongoDashboardRepository)(nil)
