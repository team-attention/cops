package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bytedance/sonic"
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
		sessionRecordsColl: db.Collection(mongoschema.RecordCollectionName),
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
			"totalInputTokens":     bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
			"totalOutputTokens":    bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
			"totalCacheReadTokens": bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
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
		bson.M{"$group": bson.M{"_id": "$" + mongoschema.RecordSessionIDField}},
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
			"from": mongoschema.RecordCollectionName,
			"let":  bson.M{"projectId": "$_id"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{
					"$expr": bson.M{"$eq": bson.A{"$" + mongoschema.RecordProjectIDField, "$$projectId"}},
				}},
				bson.M{"$group": bson.M{
					"_id":          nil,
					"sessionIds":   bson.M{"$addToSet": "$" + mongoschema.RecordSessionIDField},
					"lastActivity": bson.M{"$max": "$" + mongoschema.RecordTimestampField},
					mongoschema.MessageUsageInputTokensPath:     bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
					mongoschema.MessageUsageOutputTokensPath:    bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
					mongoschema.MessageUsageCacheReadTokensPath: bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
				}},
				bson.M{"$project": bson.M{
					"_id":          0,
					"sessionCount": bson.M{"$size": "$sessionIds"},
					"lastActivity": 1,
					mongoschema.MessageUsageInputTokensPath:     1,
					mongoschema.MessageUsageOutputTokensPath:    1,
					mongoschema.MessageUsageCacheReadTokensPath: 1,
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
			mongoschema.MessageUsageInputTokensPath:     bson.M{"$ifNull": bson.A{"$stats." + mongoschema.MessageUsageInputTokensPath, 0}},
			mongoschema.MessageUsageOutputTokensPath:    bson.M{"$ifNull": bson.A{"$stats." + mongoschema.MessageUsageOutputTokensPath, 0}},
			mongoschema.MessageUsageCacheReadTokensPath: bson.M{"$ifNull": bson.A{"$stats." + mongoschema.MessageUsageCacheReadTokensPath, 0}},
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
					TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.MessageUsageInputTokensPath),
					TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.MessageUsageOutputTokensPath),
					TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.MessageUsageCacheReadTokensPath),
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
	// Build aggregation pipeline
	pipeline := bson.A{}

	// Only filter by project if projectID is provided
	if params.Query.ProjectID != "" {
		projectOID, err := bson.ObjectIDFromHex(params.Query.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("invalid project ID: %w", err)
		}
		pipeline = append(pipeline, bson.M{"$match": bson.M{mongoschema.RecordProjectIDField: projectOID}})
	}

	// Add group stage
	pipeline = append(pipeline, bson.M{"$group": bson.M{
		"_id":                                   "$" + mongoschema.RecordSessionIDField,
		"messageCount":                          bson.M{"$sum": 1},
		"startedAt":                             bson.M{"$min": "$" + mongoschema.RecordTimestampField},
		"endedAt":                               bson.M{"$max": "$" + mongoschema.RecordTimestampField},
		mongoschema.RecordProjectIDField: bson.M{"$first": "$" + mongoschema.RecordProjectIDField},
		mongoschema.RecordGitBranchField: bson.M{"$first": "$" + mongoschema.RecordGitBranchField},
		mongoschema.MessageUsageInputTokensPath:     bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
		mongoschema.MessageUsageOutputTokensPath:    bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
		mongoschema.MessageUsageCacheReadTokensPath: bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
	}})

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
				ProjectID: doc[mongoschema.RecordProjectIDField].(bson.ObjectID).Hex(),
				GitBranch: mongoutil.Get[string](doc, mongoschema.RecordGitBranchField),
				StartedAt: mongoutil.Get[time.Time](doc, "startedAt"),
				EndedAt:   mongoutil.Get[time.Time](doc, "endedAt"),
			},
			MessageCount: mongoutil.Get[int32](doc, "messageCount"),
			Usage: repository.TokenUsageSummary{
				TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.MessageUsageInputTokensPath),
				TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.MessageUsageOutputTokensPath),
				TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.MessageUsageCacheReadTokensPath),
			},
		}

		sessions = append(sessions, session)
	}

	return structure.NewPaginatedResult(sessions, params.Page, params.PageSize, totalCount), nil
}

// GetSession retrieves detailed session information with all records.
func (r *MongoDashboardRepository) GetSession(ctx context.Context, sessionID string) (*repository.SessionDetail, error) {
	// Find all records for this session
	cursor, err := r.sessionRecordsColl.Find(ctx, bson.M{mongoschema.RecordSessionIDField: sessionID}, options.Find().SetSort(bson.M{mongoschema.RecordTimestampField: 1}))
	if err != nil {
		r.logger.Error("failed to find session records", slog.String("sessionID", sessionID), slog.Any("error", err))
		return nil, fmt.Errorf("failed to find session records: %w", err)
	}
	defer cursor.Close(ctx)

	var records []shareddomain.Record
	var detail *repository.SessionDetail

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		// Initialize detail from first record
		if detail == nil {
			projectID := ""
			if oid, ok := doc[mongoschema.RecordProjectIDField].(bson.ObjectID); ok {
				projectID = oid.Hex()
			}

			detail = &repository.SessionDetail{
				SessionBase: repository.SessionBase{
					ID:        sessionID,
					ProjectID: projectID,
					GitBranch: mongoutil.Get[string](doc, mongoschema.RecordGitBranchField),
				},
				Version: mongoutil.Get[string](doc, mongoschema.RecordVersionField),
			}
		}

		// Reconstruct domain.Record from document
		// Read "type" field to determine RecordType
		recordType := shareddomain.RecordType(mongoutil.Get[string](doc, mongoschema.RecordTypeField))

		// Marshal document back to JSON
		docBytes, err := sonic.Marshal(doc)
		if err != nil {
			r.logger.Error("failed to marshal document to JSON",
				slog.String("sessionID", sessionID),
				slog.Any("error", err))
			continue
		}

		// Unmarshal JSON into domain.Record (uses custom UnmarshalJSON)
		var record shareddomain.Record
		if err := sonic.Unmarshal(docBytes, &record); err != nil {
			r.logger.Error("failed to unmarshal record",
				slog.String("sessionID", sessionID),
				slog.String("type", string(recordType)),
				slog.Any("error", err))
			continue
		}

		records = append(records, record)
	}

	if detail == nil {
		return nil, fmt.Errorf("session not found")
	}

	detail.Records = records

	// Calculate aggregated usage and timestamps from records
	if len(records) > 0 {
		// Extract timestamps from first and last records
		if firstUserRec, ok := records[0].Data.(*shareddomain.UserRecord); ok {
			detail.StartedAt = firstUserRec.Timestamp
		} else if firstAssistantRec, ok := records[0].Data.(*shareddomain.AssistantRecord); ok {
			detail.StartedAt = firstAssistantRec.Timestamp
		}

		if lastUserRec, ok := records[len(records)-1].Data.(*shareddomain.UserRecord); ok {
			detail.EndedAt = lastUserRec.Timestamp
		} else if lastAssistantRec, ok := records[len(records)-1].Data.(*shareddomain.AssistantRecord); ok {
			detail.EndedAt = lastAssistantRec.Timestamp
		}

		// Calculate token usage from assistant records
		var inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens int64
		for _, r := range records {
			if assistantRec, ok := r.Data.(*shareddomain.AssistantRecord); ok {
				inputTokens += int64(assistantRec.Message.Usage.InputTokens)
				outputTokens += int64(assistantRec.Message.Usage.OutputTokens)
				cacheCreationTokens += int64(assistantRec.Message.Usage.CacheCreationInputTokens)
				cacheReadTokens += int64(assistantRec.Message.Usage.CacheReadInputTokens)
			}
		}

		detail.Usage = repository.TokenUsageSummary{
			TotalInputTokens:         inputTokens,
			TotalOutputTokens:        outputTokens,
			TotalCacheCreationTokens: cacheCreationTokens,
			TotalCacheReadTokens:     cacheReadTokens,
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
			"from":         mongoschema.RecordCollectionName,
			"localField":   "_id",
			"foreignField": mongoschema.RecordProjectIDField,
			"as":           "sessions",
		}},

		// Stage 2: Calculate all stats from sessions array
		bson.M{"$addFields": bson.M{
			"lastActivity": bson.M{"$max": "$sessions." + mongoschema.RecordTimestampField},
			"sessionCount": bson.M{"$size": bson.M{
				"$setUnion": bson.A{"$sessions." + mongoschema.RecordSessionIDField, bson.A{}},
			}},
			mongoschema.MessageUsageInputTokensPath:     bson.M{"$sum": "$sessions." + mongoschema.MessageUsageInputTokensPath},
			mongoschema.MessageUsageOutputTokensPath:    bson.M{"$sum": "$sessions." + mongoschema.MessageUsageOutputTokensPath},
			mongoschema.MessageUsageCacheReadTokensPath: bson.M{"$sum": "$sessions." + mongoschema.MessageUsageCacheReadTokensPath},
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
					TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.MessageUsageInputTokensPath),
					TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.MessageUsageOutputTokensPath),
					TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.MessageUsageCacheReadTokensPath),
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
			"_id":                                   "$" + mongoschema.RecordSessionIDField,
			"messageCount":                          bson.M{"$sum": 1},
			"startedAt":                             bson.M{"$min": "$" + mongoschema.RecordTimestampField},
			"endedAt":                               bson.M{"$max": "$" + mongoschema.RecordTimestampField},
			mongoschema.RecordProjectIDField: bson.M{"$first": "$" + mongoschema.RecordProjectIDField},
			mongoschema.RecordGitBranchField: bson.M{"$first": "$" + mongoschema.RecordGitBranchField},
			mongoschema.MessageUsageInputTokensPath:     bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
			mongoschema.MessageUsageOutputTokensPath:    bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
			mongoschema.MessageUsageCacheReadTokensPath: bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
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
		if oid, ok := doc[mongoschema.RecordProjectIDField].(bson.ObjectID); ok {
			projectID = oid.Hex()
		}

		session := repository.SessionSummary{
			SessionBase: repository.SessionBase{
				ID:        mongoutil.Get[string](doc, "_id"),
				ProjectID: projectID,
				GitBranch: mongoutil.Get[string](doc, mongoschema.RecordGitBranchField),
				StartedAt: mongoutil.Get[time.Time](doc, "startedAt"),
				EndedAt:   mongoutil.Get[time.Time](doc, "endedAt"),
			},
			MessageCount: mongoutil.Get[int32](doc, "messageCount"),
			Usage: repository.TokenUsageSummary{
				TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.MessageUsageInputTokensPath),
				TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.MessageUsageOutputTokensPath),
				TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.MessageUsageCacheReadTokensPath),
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
		bson.M{"$match": bson.M{mongoschema.RecordProjectIDField: projectOID}},
		bson.M{"$group": bson.M{
			"_id":          nil,
			"sessionCount": bson.M{"$addToSet": "$" + mongoschema.RecordSessionIDField},
			"lastActivity": bson.M{"$max": "$" + mongoschema.RecordTimestampField},
			mongoschema.MessageUsageInputTokensPath:     bson.M{"$sum": "$" + mongoschema.MessageUsageInputTokensPath},
			mongoschema.MessageUsageOutputTokensPath:    bson.M{"$sum": "$" + mongoschema.MessageUsageOutputTokensPath},
			mongoschema.MessageUsageCacheReadTokensPath: bson.M{"$sum": "$" + mongoschema.MessageUsageCacheReadTokensPath},
		}},
		bson.M{"$project": bson.M{
			"sessionCount": bson.M{"$size": "$sessionCount"},
			"lastActivity": 1,
			mongoschema.MessageUsageInputTokensPath:     1,
			mongoschema.MessageUsageOutputTokensPath:    1,
			mongoschema.MessageUsageCacheReadTokensPath: 1,
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
				TotalInputTokens:     mongoutil.Get[int64](doc, mongoschema.MessageUsageInputTokensPath),
				TotalOutputTokens:    mongoutil.Get[int64](doc, mongoschema.MessageUsageOutputTokensPath),
				TotalCacheReadTokens: mongoutil.Get[int64](doc, mongoschema.MessageUsageCacheReadTokensPath),
			},
		},
	}, nil
}

// Compile-time interface verification.
var _ repository.DashboardRepositoryPort = (*MongoDashboardRepository)(nil)
