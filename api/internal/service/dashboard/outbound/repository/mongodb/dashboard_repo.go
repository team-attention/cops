package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/team-attention/cops/api/internal/service/dashboard/outbound/repository"
	shareddomain "github.com/team-attention/cops/shared/domain"
)

const (
	projectsCollection      = "projects"
	sessionRecordsCollection = "session_records"
)

// Adapter implements DashboardRepositoryPort using MongoDB.
type Adapter struct {
	logger            *slog.Logger
	db                *mongo.Database
	projectsColl      *mongo.Collection
	sessionRecordsColl *mongo.Collection
}

// NewAdapter creates a new MongoDB dashboard repository adapter.
func NewAdapter(l *slog.Logger, db *mongo.Database) *Adapter {
	return &Adapter{
		logger:            l.With(slog.String("name", "dashboard.repository.mongodb")),
		db:                db,
		projectsColl:      db.Collection(projectsCollection),
		sessionRecordsColl: db.Collection(sessionRecordsCollection),
	}
}

// GetOverviewStats retrieves dashboard overview statistics.
func (a *Adapter) GetOverviewStats(ctx context.Context) (*repository.OverviewStats, error) {
	stats := &repository.OverviewStats{}

	// Get total usage from session records
	usagePipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id": nil,
			"totalInputTokens": bson.M{"$sum": "$input_tokens"},
			"totalOutputTokens": bson.M{"$sum": "$output_tokens"},
			"totalCacheReadTokens": bson.M{"$sum": "$cache_read_tokens"},
		}},
	}

	usageCursor, err := a.sessionRecordsColl.Aggregate(ctx, usagePipeline)
	if err != nil {
		a.logger.Error("failed to aggregate total usage", slog.Any("error", err))
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
	projectCount, err := a.projectsColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		a.logger.Error("failed to count projects", slog.Any("error", err))
		return nil, fmt.Errorf("failed to count projects: %w", err)
	}
	stats.ProjectCount = int32(projectCount)

	// Get session count (distinct session_ids)
	sessionPipeline := bson.A{
		bson.M{"$group": bson.M{"_id": "$session_id"}},
		bson.M{"$count": "count"},
	}

	sessionCursor, err := a.sessionRecordsColl.Aggregate(ctx, sessionPipeline)
	if err != nil {
		a.logger.Error("failed to count sessions", slog.Any("error", err))
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
	recentProjects, err := a.getRecentProjects(ctx, 5)
	if err != nil {
		a.logger.Error("failed to get recent projects", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get recent projects: %w", err)
	}
	stats.RecentProjects = recentProjects

	// Get recent sessions (top 5 by started_at)
	recentSessions, err := a.getRecentSessions(ctx, 5)
	if err != nil {
		a.logger.Error("failed to get recent sessions", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get recent sessions: %w", err)
	}
	stats.RecentSessions = recentSessions

	return stats, nil
}

// ListProjects retrieves a paginated list of projects.
func (a *Adapter) ListProjects(ctx context.Context, params repository.ListProjectsParams) (*repository.PaginatedProjects, error) {
	// Build filter
	filter := bson.M{}
	if params.Search != "" {
		filter["name"] = bson.M{"$regex": params.Search, "$options": "i"}
	}

	// Get total count
	totalCount, err := a.projectsColl.CountDocuments(ctx, filter)
	if err != nil {
		a.logger.Error("failed to count projects", slog.Any("error", err))
		return nil, fmt.Errorf("failed to count projects: %w", err)
	}

	// Calculate pagination
	totalPages := int32(math.Ceil(float64(totalCount) / float64(params.PageSize)))
	skip := int64((params.Page - 1) * params.PageSize)

	// Build sort
	sortField := "name"
	if params.SortBy != "" {
		sortField = params.SortBy
	}
	sortOrder := 1
	if params.SortDesc {
		sortOrder = -1
	}

	// Query projects
	opts := options.Find().
		SetSort(bson.M{sortField: sortOrder}).
		SetSkip(skip).
		SetLimit(int64(params.PageSize))

	cursor, err := a.projectsColl.Find(ctx, filter, opts)
	if err != nil {
		a.logger.Error("failed to find projects", slog.Any("error", err))
		return nil, fmt.Errorf("failed to find projects: %w", err)
	}
	defer cursor.Close(ctx)

	var projects []repository.ProjectSummary
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}

		project := repository.ProjectSummary{
			ID:        doc["_id"].(bson.ObjectID).Hex(),
			Name:      getString(doc, "name"),
			Path:      getString(doc, "path"),
			GitBranch: getString(doc, "git_branch"),
		}

		// Get session stats for this project
		stats, err := a.getProjectStats(ctx, project.ID)
		if err == nil {
			project.SessionCount = stats.SessionCount
			project.Usage = stats.Usage
			project.LastActivity = stats.LastActivity
		}

		projects = append(projects, project)
	}

	return &repository.PaginatedProjects{
		Projects:    projects,
		CurrentPage: params.Page,
		PageSize:    params.PageSize,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
	}, nil
}

// GetProject retrieves detailed project information.
func (a *Adapter) GetProject(ctx context.Context, projectID string) (*repository.ProjectDetail, error) {
	objectID, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	var doc bson.M
	err = a.projectsColl.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("project not found")
		}
		a.logger.Error("failed to find project", slog.String("projectID", projectID), slog.Any("error", err))
		return nil, fmt.Errorf("failed to find project: %w", err)
	}

	detail := &repository.ProjectDetail{
		ID:        projectID,
		Name:      getString(doc, "name"),
		Path:      getString(doc, "path"),
		GitBranch: getString(doc, "git_branch"),
		Worktrees: getStringArray(doc, "worktrees"),
		CreatedAt: getTime(doc, "registered_at"),
	}

	// Get session stats
	stats, err := a.getProjectStats(ctx, projectID)
	if err == nil {
		detail.SessionCount = stats.SessionCount
		detail.Usage = stats.Usage
		detail.LastActivity = stats.LastActivity
	}

	return detail, nil
}

// ListSessions retrieves paginated sessions for a project.
func (a *Adapter) ListSessions(ctx context.Context, params repository.ListSessionsParams) (*repository.PaginatedSessions, error) {
	projectOID, err := bson.ObjectIDFromHex(params.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}

	// Aggregate sessions from session_records
	pipeline := bson.A{
		bson.M{"$match": bson.M{"project_id": projectOID}},
		bson.M{"$group": bson.M{
			"_id":          "$session_id",
			"projectId":    bson.M{"$first": "$project_id"},
			"gitBranch":    bson.M{"$first": "$git_branch"},
			"messageCount": bson.M{"$sum": 1},
			"startedAt":    bson.M{"$min": "$timestamp"},
			"endedAt":      bson.M{"$max": "$timestamp"},
			"inputTokens":  bson.M{"$sum": "$input_tokens"},
			"outputTokens": bson.M{"$sum": "$output_tokens"},
			"cacheReadTokens": bson.M{"$sum": "$cache_read_tokens"},
		}},
	}

	// Add sort
	sortField := "startedAt"
	if params.SortBy != "" {
		sortField = params.SortBy
	}
	sortOrder := -1
	if !params.SortDesc {
		sortOrder = 1
	}
	pipeline = append(pipeline, bson.M{"$sort": bson.M{sortField: sortOrder}})

	// Get total count first
	countPipeline := append(pipeline, bson.M{"$count": "count"})
	countCursor, err := a.sessionRecordsColl.Aggregate(ctx, countPipeline)
	if err != nil {
		a.logger.Error("failed to count sessions", slog.Any("error", err))
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
	skip := int64((params.Page - 1) * params.PageSize)
	pipeline = append(pipeline,
		bson.M{"$skip": skip},
		bson.M{"$limit": int64(params.PageSize)},
	)

	cursor, err := a.sessionRecordsColl.Aggregate(ctx, pipeline)
	if err != nil {
		a.logger.Error("failed to aggregate sessions", slog.Any("error", err))
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
			ID:           getString(doc, "_id"),
			ProjectID:    doc["projectId"].(bson.ObjectID).Hex(),
			GitBranch:    getString(doc, "gitBranch"),
			MessageCount: getInt32(doc, "messageCount"),
			StartedAt:    getTime(doc, "startedAt"),
			EndedAt:      getTime(doc, "endedAt"),
			Usage: repository.TokenUsageSummary{
				TotalInputTokens:     getInt64(doc, "inputTokens"),
				TotalOutputTokens:    getInt64(doc, "outputTokens"),
				TotalCacheReadTokens: getInt64(doc, "cacheReadTokens"),
			},
		}

		sessions = append(sessions, session)
	}

	totalPages := int32(math.Ceil(float64(totalCount) / float64(params.PageSize)))

	return &repository.PaginatedSessions{
		Sessions:    sessions,
		CurrentPage: params.Page,
		PageSize:    params.PageSize,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
	}, nil
}

// GetSession retrieves detailed session information with all records.
func (a *Adapter) GetSession(ctx context.Context, sessionID string) (*repository.SessionDetail, error) {
	// Find all records for this session
	cursor, err := a.sessionRecordsColl.Find(ctx, bson.M{"session_id": sessionID}, options.Find().SetSort(bson.M{"timestamp": 1}))
	if err != nil {
		a.logger.Error("failed to find session records", slog.String("sessionID", sessionID), slog.Any("error", err))
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
			if oid, ok := doc["project_id"].(bson.ObjectID); ok {
				projectID = oid.Hex()
			}

			detail = &repository.SessionDetail{
				ID:        sessionID,
				ProjectID: projectID,
				GitBranch: getString(doc, "git_branch"),
				CWD:       getString(doc, "cwd"),
				Version:   getString(doc, "version"),
			}
		}

		// Convert to domain SessionRecord
		record := shareddomain.SessionRecord{
			UUID:        getString(doc, "uuid"),
			ParentUUID:  getString(doc, "parent_uuid"),
			SessionID:   getString(doc, "session_id"),
			Type:        shareddomain.SessionType(getString(doc, "type")),
			Timestamp:   getTime(doc, "timestamp"),
			CWD:         getString(doc, "cwd"),
			GitBranch:   getString(doc, "git_branch"),
			Version:     getString(doc, "version"),
			UserType:    getString(doc, "user_type"),
			IsSidechain: getBool(doc, "is_sidechain"),
			IsMeta:      getBool(doc, "is_meta"),
			Slug:        getString(doc, "slug"),
			RequestID:   getString(doc, "request_id"),
		}

		// Add usage if available
		if getInt(doc, "input_tokens") > 0 || getInt(doc, "output_tokens") > 0 {
			record.Message = &shareddomain.Message{
				Usage: &shareddomain.Usage{
					InputTokens:          getInt(doc, "input_tokens"),
					OutputTokens:         getInt(doc, "output_tokens"),
					CacheReadInputTokens: getInt(doc, "cache_read_tokens"),
				},
			}
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

func (a *Adapter) getRecentProjects(ctx context.Context, limit int) ([]repository.ProjectSummary, error) {
	// Get projects sorted by last activity
	pipeline := bson.A{
		bson.M{"$lookup": bson.M{
			"from": sessionRecordsCollection,
			"localField": "_id",
			"foreignField": "project_id",
			"as": "sessions",
		}},
		bson.M{"$addFields": bson.M{
			"lastActivity": bson.M{"$max": "$sessions.timestamp"},
		}},
		bson.M{"$sort": bson.M{"lastActivity": -1}},
		bson.M{"$limit": limit},
	}

	cursor, err := a.projectsColl.Aggregate(ctx, pipeline)
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

		projectID := doc["_id"].(bson.ObjectID).Hex()
		project := repository.ProjectSummary{
			ID:           projectID,
			Name:         getString(doc, "name"),
			Path:         getString(doc, "path"),
			GitBranch:    getString(doc, "git_branch"),
			LastActivity: getTime(doc, "lastActivity"),
		}

		// Get detailed stats
		stats, err := a.getProjectStats(ctx, projectID)
		if err == nil {
			project.SessionCount = stats.SessionCount
			project.Usage = stats.Usage
		}

		projects = append(projects, project)
	}

	return projects, nil
}

func (a *Adapter) getRecentSessions(ctx context.Context, limit int) ([]repository.SessionSummary, error) {
	pipeline := bson.A{
		bson.M{"$group": bson.M{
			"_id":          "$session_id",
			"projectId":    bson.M{"$first": "$project_id"},
			"gitBranch":    bson.M{"$first": "$git_branch"},
			"messageCount": bson.M{"$sum": 1},
			"startedAt":    bson.M{"$min": "$timestamp"},
			"endedAt":      bson.M{"$max": "$timestamp"},
			"inputTokens":  bson.M{"$sum": "$input_tokens"},
			"outputTokens": bson.M{"$sum": "$output_tokens"},
			"cacheReadTokens": bson.M{"$sum": "$cache_read_tokens"},
		}},
		bson.M{"$sort": bson.M{"startedAt": -1}},
		bson.M{"$limit": limit},
	}

	cursor, err := a.sessionRecordsColl.Aggregate(ctx, pipeline)
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
		if oid, ok := doc["projectId"].(bson.ObjectID); ok {
			projectID = oid.Hex()
		}

		session := repository.SessionSummary{
			ID:           getString(doc, "_id"),
			ProjectID:    projectID,
			GitBranch:    getString(doc, "gitBranch"),
			MessageCount: getInt32(doc, "messageCount"),
			StartedAt:    getTime(doc, "startedAt"),
			EndedAt:      getTime(doc, "endedAt"),
			Usage: repository.TokenUsageSummary{
				TotalInputTokens:     getInt64(doc, "inputTokens"),
				TotalOutputTokens:    getInt64(doc, "outputTokens"),
				TotalCacheReadTokens: getInt64(doc, "cacheReadTokens"),
			},
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (a *Adapter) getProjectStats(ctx context.Context, projectID string) (*repository.ProjectSummary, error) {
	projectOID, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return nil, err
	}

	pipeline := bson.A{
		bson.M{"$match": bson.M{"project_id": projectOID}},
		bson.M{"$group": bson.M{
			"_id":             nil,
			"sessionCount":    bson.M{"$addToSet": "$session_id"},
			"lastActivity":    bson.M{"$max": "$timestamp"},
			"inputTokens":     bson.M{"$sum": "$input_tokens"},
			"outputTokens":    bson.M{"$sum": "$output_tokens"},
			"cacheReadTokens": bson.M{"$sum": "$cache_read_tokens"},
		}},
		bson.M{"$project": bson.M{
			"sessionCount": bson.M{"$size": "$sessionCount"},
			"lastActivity": 1,
			"inputTokens": 1,
			"outputTokens": 1,
			"cacheReadTokens": 1,
		}},
	}

	cursor, err := a.sessionRecordsColl.Aggregate(ctx, pipeline)
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
		SessionCount: getInt32(doc, "sessionCount"),
		LastActivity: getTime(doc, "lastActivity"),
		Usage: repository.TokenUsageSummary{
			TotalInputTokens:     getInt64(doc, "inputTokens"),
			TotalOutputTokens:    getInt64(doc, "outputTokens"),
			TotalCacheReadTokens: getInt64(doc, "cacheReadTokens"),
		},
	}, nil
}

// Helper functions for type conversions

func getString(m bson.M, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getStringArray(m bson.M, key string) []string {
	if v, ok := m[key]; ok {
		if arr, ok := v.(bson.A); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

func getInt(m bson.M, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int32:
			return int(val)
		case int64:
			return int(val)
		}
	}
	return 0
}

func getInt32(m bson.M, key string) int32 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int32:
			return val
		case int:
			return int32(val)
		case int64:
			return int32(val)
		}
	}
	return 0
}

func getInt64(m bson.M, key string) int64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int64:
			return val
		case int:
			return int64(val)
		case int32:
			return int64(val)
		}
	}
	return 0
}

func getBool(m bson.M, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getTime(m bson.M, key string) time.Time {
	if v, ok := m[key]; ok {
		if t, ok := v.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}

// Compile-time interface verification.
var _ repository.DashboardRepositoryPort = (*Adapter)(nil)
