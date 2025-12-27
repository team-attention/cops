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
)

// MongoDashboardRepository implements DashboardRepositoryPort using MongoDB.
type MongoDashboardRepository struct {
	logger             *slog.Logger
	projectsColl       *mongo.Collection
	sessionRecordsColl *mongo.Collection
}

// NewMongoDashboardRepository creates a new MongoDB dashboard repository adapter.
func NewMongoDashboardRepository(l *slog.Logger, db *mongo.Database) *MongoDashboardRepository {
	return &MongoDashboardRepository{
		logger:             l.With(slog.String("name", "dashboard.repository.mongodb")),
		projectsColl:       db.Collection(mongoschema.ProjectCollectionName),
		sessionRecordsColl: db.Collection(mongoschema.SessionRecordCollectionName),
	}
}

// GetOverviewStats retrieves dashboard overview statistics.
func (r *MongoDashboardRepository) GetOverviewStats(ctx context.Context) (*repository.OverviewStats, error) {
	r.logger.Debug("GetOverviewStats called")
	stats := &repository.OverviewStats{}

	// Get total usage from session records
	usagePipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id":                  nil,
			"totalInputTokens":     bson.M{"$sum": "$" + mongoschema.SessionRecordInputTokensField},
			"totalOutputTokens":    bson.M{"$sum": "$" + mongoschema.SessionRecordOutputTokensField},
			"totalCacheReadTokens": bson.M{"$sum": "$" + mongoschema.SessionRecordCacheReadTokensField},
		}},
	}

	usageCursor, err := r.sessionRecordsColl.Aggregate(ctx, usagePipeline)
	if err != nil {
		r.logger.Error("failed to aggregate total usage", slog.Any("error", err))
		return nil, fmt.Errorf("failed to aggregate total usage: %w", err)
	}
	defer usageCursor.Close(ctx)

	if usageCursor.Next(ctx) {
		var result struct {
			TotalInputTokens     int64 `bson:"totalInputTokens"`
			TotalOutputTokens    int64 `bson:"totalOutputTokens"`
			TotalCacheReadTokens int64 `bson:"totalCacheReadTokens"`
		}
		if err := usageCursor.Decode(&result); err == nil {
			stats.TotalUsage = repository.TokenUsageSummary{
				TotalInputTokens:     result.TotalInputTokens,
				TotalOutputTokens:    result.TotalOutputTokens,
				TotalCacheReadTokens: result.TotalCacheReadTokens,
			}
		}
	}

	// Get project count
	projectCount, err := r.projectsColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		r.logger.Error("failed to count projects", slog.Any("error", err))
		return nil, fmt.Errorf("failed to count projects: %w", err)
	}
	stats.ProjectCount = int32(projectCount)
	r.logger.Debug("overview stats counts",
		slog.Int64("projectCount", projectCount),
	)

	// Get session count (distinct session_ids)
	sessionPipeline := bson.A{
		bson.M{"$group": bson.M{"_id": "$" + mongoschema.SessionRecordSessionIDField}},
		bson.M{"$count": "count"},
	}

	sessionCursor, err := r.sessionRecordsColl.Aggregate(ctx, sessionPipeline)
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

	// Get recent projects (top 5 by last activity)
	recentProjects, err := r.getRecentProjects(ctx, 5)
	if err != nil {
		r.logger.Error("failed to get recent projects", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get recent projects: %w", err)
	}
	stats.RecentProjects = recentProjects

	// Get recent sessions (top 5 by started_at)
	recentSessions, err := r.getRecentSessions(ctx, 5)
	if err != nil {
		r.logger.Error("failed to get recent sessions", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get recent sessions: %w", err)
	}
	stats.RecentSessions = recentSessions

	return stats, nil
}

// ListProjects retrieves a paginated list of projects.
func (r *MongoDashboardRepository) ListProjects(ctx context.Context, params repository.ListProjectsParams) (*repository.PaginatedProjects, error) {
	r.logger.Debug("ListProjects called",
		slog.Int("page", int(params.Page)),
		slog.Int("pageSize", int(params.PageSize)),
	)
	// Single aggregation pipeline with $facet for count and data
	pipeline := bson.A{
		// Stage 1: Lookup session records with sub-pipeline for aggregation
		bson.M{"$lookup": bson.M{
			"from": mongoschema.SessionRecordCollectionName,
			"let":  bson.M{"projectId": "$_id"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{
					"$expr": bson.M{"$eq": bson.A{"$" + mongoschema.SessionRecordProjectIDField, "$$projectId"}},
				}},
				bson.M{"$group": bson.M{
					"_id":          nil,
					"sessionIds":   bson.M{"$addToSet": "$" + mongoschema.SessionRecordSessionIDField},
					"lastActivity": bson.M{"$max": "$" + mongoschema.SessionRecordTimestampField},
					mongoschema.SessionRecordInputTokensField:     bson.M{"$sum": "$" + mongoschema.SessionRecordInputTokensField},
					mongoschema.SessionRecordOutputTokensField:    bson.M{"$sum": "$" + mongoschema.SessionRecordOutputTokensField},
					mongoschema.SessionRecordCacheReadTokensField: bson.M{"$sum": "$" + mongoschema.SessionRecordCacheReadTokensField},
				}},
				bson.M{"$project": bson.M{
					"_id":          0,
					"sessionCount": bson.M{"$size": "$sessionIds"},
					"lastActivity": 1,
					mongoschema.SessionRecordInputTokensField:     1,
					mongoschema.SessionRecordOutputTokensField:    1,
					mongoschema.SessionRecordCacheReadTokensField: 1,
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
			mongoschema.ProjectIDField:                    1,
			mongoschema.ProjectNameField:                  1,
			mongoschema.ProjectPathField:                  1,
			mongoschema.ProjectGitBranchField:             1,
			"sessionCount":                                bson.M{"$ifNull": bson.A{"$stats.sessionCount", 0}},
			"lastActivity":                                bson.M{"$ifNull": bson.A{"$stats.lastActivity", nil}},
			mongoschema.SessionRecordInputTokensField:     bson.M{"$ifNull": bson.A{"$stats." + mongoschema.SessionRecordInputTokensField, 0}},
			mongoschema.SessionRecordOutputTokensField:    bson.M{"$ifNull": bson.A{"$stats." + mongoschema.SessionRecordOutputTokensField, 0}},
			mongoschema.SessionRecordCacheReadTokensField: bson.M{"$ifNull": bson.A{"$stats." + mongoschema.SessionRecordCacheReadTokensField, 0}},
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
					TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.SessionRecordInputTokensField),
					TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.SessionRecordOutputTokensField),
					TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.SessionRecordCacheReadTokensField),
				},
			},
		}
		projects = append(projects, project)
	}

	return structure.NewPaginatedResult(projects, params.Page, params.PageSize, totalCount), nil
}

// GetProject retrieves detailed project information.
func (r *MongoDashboardRepository) GetProject(ctx context.Context, projectID string) (*repository.ProjectDetail, error) {
	objectID, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	var doc bson.M
	err = r.projectsColl.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
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
			ClaudeDir:    mongoutil.Get[string](doc, mongoschema.ProjectClaudeDirField),
			Worktrees:    mongoutil.GetSlice[string](doc, mongoschema.ProjectWorktreesField),
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

// ListSessions retrieves paginated sessions for a project.
func (r *MongoDashboardRepository) ListSessions(ctx context.Context, params repository.ListSessionsParams) (*repository.PaginatedSessions, error) {
	projectOID, err := bson.ObjectIDFromHex(params.Query.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	// Aggregate sessions from session_records
	pipeline := bson.A{
		bson.M{"$match": bson.M{mongoschema.SessionRecordProjectIDField: projectOID}},
		bson.M{"$group": bson.M{
			"_id":                                   "$" + mongoschema.SessionRecordSessionIDField,
			"messageCount":                          bson.M{"$sum": 1},
			"startedAt":                             bson.M{"$min": "$" + mongoschema.SessionRecordTimestampField},
			"endedAt":                               bson.M{"$max": "$" + mongoschema.SessionRecordTimestampField},
			mongoschema.SessionRecordProjectIDField: bson.M{"$first": "$" + mongoschema.SessionRecordProjectIDField},
			mongoschema.SessionRecordGitBranchField: bson.M{"$first": "$" + mongoschema.SessionRecordGitBranchField},
			mongoschema.SessionRecordInputTokensField:     bson.M{"$sum": "$" + mongoschema.SessionRecordInputTokensField},
			mongoschema.SessionRecordOutputTokensField:    bson.M{"$sum": "$" + mongoschema.SessionRecordOutputTokensField},
			mongoschema.SessionRecordCacheReadTokensField: bson.M{"$sum": "$" + mongoschema.SessionRecordCacheReadTokensField},
		}},
	}

	// Add sort
	sortField := "startedAt"
	if params.Query.SortBy != "" {
		sortField = params.Query.SortBy
	}
	sortOrder := -1
	if !params.Query.SortDesc {
		sortOrder = 1
	}
	pipeline = append(pipeline, bson.M{"$sort": bson.M{sortField: sortOrder}})

	// Get total count first
	countPipeline := append(pipeline, bson.M{"$count": "count"})
	countCursor, err := r.sessionRecordsColl.Aggregate(ctx, countPipeline)
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

	cursor, err := r.sessionRecordsColl.Aggregate(ctx, pipeline)
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
				ProjectID: doc[mongoschema.SessionRecordProjectIDField].(bson.ObjectID).Hex(),
				GitBranch: mongoutil.Get[string](doc, mongoschema.SessionRecordGitBranchField),
				StartedAt: mongoutil.Get[time.Time](doc, "startedAt"),
				EndedAt:   mongoutil.Get[time.Time](doc, "endedAt"),
			},
			MessageCount: mongoutil.Get[int32](doc, "messageCount"),
			Usage: repository.TokenUsageSummary{
				TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.SessionRecordInputTokensField),
				TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.SessionRecordOutputTokensField),
				TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.SessionRecordCacheReadTokensField),
			},
		}

		sessions = append(sessions, session)
	}

	return structure.NewPaginatedResult(sessions, params.Page, params.PageSize, totalCount), nil
}

// GetSession retrieves detailed session information with all records.
func (r *MongoDashboardRepository) GetSession(ctx context.Context, sessionID string) (*repository.SessionDetail, error) {
	// Find all records for this session
	cursor, err := r.sessionRecordsColl.Find(ctx, bson.M{mongoschema.SessionRecordSessionIDField: sessionID}, options.Find().SetSort(bson.M{mongoschema.SessionRecordTimestampField: 1}))
	if err != nil {
		r.logger.Error("failed to find session records", slog.String("sessionID", sessionID), slog.Any("error", err))
		return nil, fmt.Errorf("failed to find session records: %w", err)
	}
	defer cursor.Close(ctx)

	var records []shareddomain.SessionRecord
	var detail *repository.SessionDetail

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		// Initialize detail from first record
		if detail == nil {
			projectID := ""
			if oid, ok := doc[mongoschema.SessionRecordProjectIDField].(bson.ObjectID); ok {
				projectID = oid.Hex()
			}

			detail = &repository.SessionDetail{
				SessionBase: repository.SessionBase{
					ID:        sessionID,
					ProjectID: projectID,
					GitBranch: mongoutil.Get[string](doc, mongoschema.SessionRecordGitBranchField),
				},
				CWD:     mongoutil.Get[string](doc, mongoschema.SessionRecordCWDField),
				Version: mongoutil.Get[string](doc, mongoschema.SessionRecordVersionField),
			}
		}

		// Convert to domain SessionRecord
		record := shareddomain.SessionRecord{
			UUID:        mongoutil.Get[string](doc, mongoschema.SessionRecordUUIDField),
			ParentUUID:  mongoutil.Get[string](doc, mongoschema.SessionRecordParentUUIDField),
			SessionID:   mongoutil.Get[string](doc, mongoschema.SessionRecordSessionIDField),
			Type:        shareddomain.SessionType(mongoutil.Get[string](doc, mongoschema.SessionRecordTypeField)),
			Timestamp:   mongoutil.Get[time.Time](doc, mongoschema.SessionRecordTimestampField),
			CWD:         mongoutil.Get[string](doc, mongoschema.SessionRecordCWDField),
			GitBranch:   mongoutil.Get[string](doc, mongoschema.SessionRecordGitBranchField),
			Version:     mongoutil.Get[string](doc, mongoschema.SessionRecordVersionField),
			UserType:    mongoutil.Get[string](doc, mongoschema.SessionRecordUserTypeField),
			IsSidechain: mongoutil.Get[bool](doc, mongoschema.SessionRecordIsSidechainField),
			IsMeta:      mongoutil.Get[bool](doc, mongoschema.SessionRecordIsMetaField),
			Slug:        mongoutil.Get[string](doc, mongoschema.SessionRecordSlugField),
			RequestID:   mongoutil.Get[string](doc, mongoschema.SessionRecordRequestIDField),
		}

		// Reconstruct message if any message field exists
		hasMessage := mongoutil.Get[string](doc, mongoschema.SessionRecordMessageIDField) != "" ||
			mongoutil.Get[string](doc, mongoschema.SessionRecordMessageRoleField) != "" ||
			mongoutil.Get[string](doc, mongoschema.SessionRecordMessageTypeField) != "" ||
			mongoutil.Get[int](doc, mongoschema.SessionRecordInputTokensField) > 0 ||
			mongoutil.Get[int](doc, mongoschema.SessionRecordOutputTokensField) > 0

		if hasMessage {
			msg := &shareddomain.Message{
				ID:         mongoutil.Get[string](doc, mongoschema.SessionRecordMessageIDField),
				Type:       mongoutil.Get[string](doc, mongoschema.SessionRecordMessageTypeField),
				Role:       mongoutil.Get[string](doc, mongoschema.SessionRecordMessageRoleField),
				Model:      mongoutil.Get[string](doc, mongoschema.SessionRecordMessageModelField),
				StopReason: mongoutil.Get[string](doc, mongoschema.SessionRecordStopReasonField),
			}

			// Reconstruct content if available (only text content is stored)
			if content := mongoutil.Get[string](doc, mongoschema.SessionRecordMessageContentField); content != "" {
				msg.Content = &shareddomain.MessageContent{
					Text:     &content,
					IsBlocks: false,
				}
			}

			// Add usage if token data exists
			if mongoutil.Get[int](doc, mongoschema.SessionRecordInputTokensField) > 0 ||
				mongoutil.Get[int](doc, mongoschema.SessionRecordOutputTokensField) > 0 ||
				mongoutil.Get[int](doc, mongoschema.SessionRecordCacheReadTokensField) > 0 {
				msg.Usage = &shareddomain.Usage{
					InputTokens:          mongoutil.Get[int](doc, mongoschema.SessionRecordInputTokensField),
					OutputTokens:         mongoutil.Get[int](doc, mongoschema.SessionRecordOutputTokensField),
					CacheReadInputTokens: mongoutil.Get[int](doc, mongoschema.SessionRecordCacheReadTokensField),
				}
			}

			record.Message = msg
		}

		records = append(records, record)
	}

	if detail == nil {
		return nil, fmt.Errorf("session not found")
	}

	detail.Records = records

	// Calculate aggregated usage and timestamps
	if len(records) > 0 {
		detail.StartedAt = records[0].Timestamp
		detail.EndedAt = records[len(records)-1].Timestamp

		var inputTokens, outputTokens, cacheReadTokens int64
		for _, r := range records {
			if r.Message != nil && r.Message.Usage != nil {
				inputTokens += int64(r.Message.Usage.InputTokens)
				outputTokens += int64(r.Message.Usage.OutputTokens)
				cacheReadTokens += int64(r.Message.Usage.CacheReadInputTokens)
			}
		}

		detail.Usage = repository.TokenUsageSummary{
			TotalInputTokens:     inputTokens,
			TotalOutputTokens:    outputTokens,
			TotalCacheReadTokens: cacheReadTokens,
		}
	}

	return detail, nil
}

// Helper methods

func (r *MongoDashboardRepository) getRecentProjects(ctx context.Context, limit int) ([]repository.ProjectSummary, error) {
	// Get projects sorted by last activity with all stats calculated in-pipeline
	pipeline := bson.A{
		// Stage 1: Lookup session records
		bson.M{"$lookup": bson.M{
			"from":         mongoschema.SessionRecordCollectionName,
			"localField":   "_id",
			"foreignField": mongoschema.SessionRecordProjectIDField,
			"as":           "sessions",
		}},

		// Stage 2: Calculate all stats from sessions array
		bson.M{"$addFields": bson.M{
			"lastActivity": bson.M{"$max": "$sessions." + mongoschema.SessionRecordTimestampField},
			"sessionCount": bson.M{"$size": bson.M{
				"$setUnion": bson.A{"$sessions." + mongoschema.SessionRecordSessionIDField, bson.A{}},
			}},
			mongoschema.SessionRecordInputTokensField:     bson.M{"$sum": "$sessions." + mongoschema.SessionRecordInputTokensField},
			mongoschema.SessionRecordOutputTokensField:    bson.M{"$sum": "$sessions." + mongoschema.SessionRecordOutputTokensField},
			mongoschema.SessionRecordCacheReadTokensField: bson.M{"$sum": "$sessions." + mongoschema.SessionRecordCacheReadTokensField},
		}},

		// Stage 3: Sort by last activity (descending)
		bson.M{"$sort": bson.M{"lastActivity": -1}},

		// Stage 4: Limit results
		bson.M{"$limit": limit},

		// Stage 5: Project to remove the sessions array (no longer needed)
		bson.M{"$project": bson.M{
			"sessions": 0,
		}},
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
					TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.SessionRecordInputTokensField),
					TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.SessionRecordOutputTokensField),
					TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.SessionRecordCacheReadTokensField),
				},
			},
		}

		projects = append(projects, project)
	}

	return projects, nil
}

func (r *MongoDashboardRepository) getRecentSessions(ctx context.Context, limit int) ([]repository.SessionSummary, error) {
	pipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id":                                   "$" + mongoschema.SessionRecordSessionIDField,
			"messageCount":                          bson.M{"$sum": 1},
			"startedAt":                             bson.M{"$min": "$" + mongoschema.SessionRecordTimestampField},
			"endedAt":                               bson.M{"$max": "$" + mongoschema.SessionRecordTimestampField},
			mongoschema.SessionRecordProjectIDField: bson.M{"$first": "$" + mongoschema.SessionRecordProjectIDField},
			mongoschema.SessionRecordGitBranchField: bson.M{"$first": "$" + mongoschema.SessionRecordGitBranchField},
			mongoschema.SessionRecordInputTokensField:     bson.M{"$sum": "$" + mongoschema.SessionRecordInputTokensField},
			mongoschema.SessionRecordOutputTokensField:    bson.M{"$sum": "$" + mongoschema.SessionRecordOutputTokensField},
			mongoschema.SessionRecordCacheReadTokensField: bson.M{"$sum": "$" + mongoschema.SessionRecordCacheReadTokensField},
		}},
		bson.M{"$sort": bson.M{"startedAt": -1}},
		bson.M{"$limit": limit},
	}

	cursor, err := r.sessionRecordsColl.Aggregate(ctx, pipeline)
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
		if oid, ok := doc[mongoschema.SessionRecordProjectIDField].(bson.ObjectID); ok {
			projectID = oid.Hex()
		}

		session := repository.SessionSummary{
			SessionBase: repository.SessionBase{
				ID:        mongoutil.Get[string](doc, "_id"),
				ProjectID: projectID,
				GitBranch: mongoutil.Get[string](doc, mongoschema.SessionRecordGitBranchField),
				StartedAt: mongoutil.Get[time.Time](doc, "startedAt"),
				EndedAt:   mongoutil.Get[time.Time](doc, "endedAt"),
			},
			MessageCount: mongoutil.Get[int32](doc, "messageCount"),
			Usage: repository.TokenUsageSummary{
				TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.SessionRecordInputTokensField),
				TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.SessionRecordOutputTokensField),
				TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.SessionRecordCacheReadTokensField),
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
		bson.M{"$match": bson.M{mongoschema.SessionRecordProjectIDField: projectOID}},
		bson.M{"$group": bson.M{
			"_id":          nil,
			"sessionCount": bson.M{"$addToSet": "$" + mongoschema.SessionRecordSessionIDField},
			"lastActivity": bson.M{"$max": "$" + mongoschema.SessionRecordTimestampField},
			mongoschema.SessionRecordInputTokensField:     bson.M{"$sum": "$" + mongoschema.SessionRecordInputTokensField},
			mongoschema.SessionRecordOutputTokensField:    bson.M{"$sum": "$" + mongoschema.SessionRecordOutputTokensField},
			mongoschema.SessionRecordCacheReadTokensField: bson.M{"$sum": "$" + mongoschema.SessionRecordCacheReadTokensField},
		}},
		bson.M{"$project": bson.M{
			"sessionCount": bson.M{"$size": "$sessionCount"},
			"lastActivity": 1,
			mongoschema.SessionRecordInputTokensField:     1,
			mongoschema.SessionRecordOutputTokensField:    1,
			mongoschema.SessionRecordCacheReadTokensField: 1,
		}},
	}

	cursor, err := r.sessionRecordsColl.Aggregate(ctx, pipeline)
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
				TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.SessionRecordInputTokensField),
				TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.SessionRecordOutputTokensField),
				TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.SessionRecordCacheReadTokensField),
			},
		},
	}, nil
}

// Compile-time interface verification.
var _ repository.DashboardRepositoryPort = (*MongoDashboardRepository)(nil)
