package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/team-attention/cops/api/internal/service/featured/outbound/repository"
	"github.com/team-attention/cops/shared/domain/mongoschema"
	session "github.com/team-attention/cops/shared/domain/v2"
)

// Aggregation output field name constants.
const (
	aggFeaturedInputTokensField         = "inputTokens"
	aggFeaturedOutputTokensField        = "outputTokens"
	aggFeaturedCacheReadTokensField     = "cacheReadTokens"
	aggFeaturedCacheCreationTokensField = "cacheCreationTokens"
	aggFeaturedPromptCountField         = "promptCount"
	aggFeaturedLastActivityField        = "lastActivityAt"
	aggFeaturedToolCountField           = "count"
)

// MongoFeaturedBoardRepository implements FeaturedBoardRepositoryPort using MongoDB.
type MongoFeaturedBoardRepository struct {
	logger       *slog.Logger
	sessionsColl *mongo.Collection
	projectsColl *mongo.Collection
}

// NewMongoFeaturedBoardRepository creates a new MongoDB featured board repository adapter.
func NewMongoFeaturedBoardRepository(l *slog.Logger, db *mongo.Database) *MongoFeaturedBoardRepository {
	return &MongoFeaturedBoardRepository{
		logger:       l.With(slog.String("name", "featured.repository.mongodb")),
		sessionsColl: db.Collection(mongoschema.SessionsCollectionName),
		projectsColl: db.Collection(mongoschema.ProjectCollectionName),
	}
}

// GetEarliestProjectForMembers returns the earliest registered project (by registeredAt ASC)
// for each member within the given organization.
// Uses the sessions collection to find which projects each member has activity in,
// then looks up the project with the earliest registeredAt in the projects collection.
func (r *MongoFeaturedBoardRepository) GetEarliestProjectForMembers(
	ctx context.Context,
	organizationID string,
	memberIDs []string,
) (map[string]string, error) {
	if len(memberIDs) == 0 {
		return map[string]string{}, nil
	}

	orgOID, err := bson.ObjectIDFromHex(organizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}

	memberOIDSet := make([]bson.ObjectID, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		oid, err := bson.ObjectIDFromHex(memberID)
		if err != nil {
			r.logger.Warn("invalid member ID, skipping",
				slog.String("memberID", memberID),
				slog.Any("error", err),
			)
			continue
		}
		memberOIDSet = append(memberOIDSet, oid)
	}

	if len(memberOIDSet) == 0 {
		return map[string]string{}, nil
	}

	// For each member, find all distinct projectIDs from sessions, then pick the earliest by registeredAt.
	// Pipeline: sessions -> group by userId -> lookup projects -> sort by registeredAt -> pick first per user.
	pipeline := bson.A{
		// Match sessions for these members
		bson.M{"$match": bson.M{
			mongoschema.SessionUserIDField: bson.M{"$in": memberOIDSet},
		}},
		// Group by userId to get distinct projectIds per user
		bson.M{"$group": bson.M{
			"_id":        "$" + mongoschema.SessionUserIDField,
			"projectIds": bson.M{"$addToSet": "$" + mongoschema.SessionProjectIDField},
		}},
		// Lookup projects for each user's projectIds
		bson.M{"$lookup": bson.M{
			"from": mongoschema.ProjectCollectionName,
			"let":  bson.M{"projectIds": "$projectIds"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{
					"$expr": bson.M{
						"$and": bson.A{
							bson.M{"$in": bson.A{"$_id", "$$projectIds"}},
							bson.M{"$eq": bson.A{"$" + mongoschema.ProjectOrganizationIDField, orgOID}},
						},
					},
				}},
				bson.M{"$sort": bson.M{mongoschema.ProjectRegisteredAtField: 1}},
				bson.M{"$limit": 1},
				bson.M{"$project": bson.M{
					mongoschema.ProjectIDField:           1,
					mongoschema.ProjectRegisteredAtField: 1,
				}},
			},
			"as": "earliestProject",
		}},
		// Filter out members with no matching project in the org
		bson.M{"$match": bson.M{
			"earliestProject": bson.M{"$ne": bson.A{}},
		}},
		// Unwind the earliest project
		bson.M{"$unwind": "$earliestProject"},
		// Project final fields
		bson.M{"$project": bson.M{
			"_id":       1,
			"projectId": "$earliestProject._id",
		}},
	}

	cursor, err := r.sessionsColl.Aggregate(ctx, pipeline)
	if err != nil {
		r.logger.Error("failed to get earliest projects for members", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get earliest projects for members: %w", err)
	}
	defer cursor.Close(ctx)

	result := make(map[string]string, len(memberIDs))
	for cursor.Next(ctx) {
		var row struct {
			UserID    bson.ObjectID `bson:"_id"`
			ProjectID bson.ObjectID `bson:"projectId"`
		}
		if err := cursor.Decode(&row); err != nil {
			r.logger.Error("failed to decode project result", slog.Any("error", err))
			return nil, fmt.Errorf("failed to decode project result: %w", err)
		}
		result[row.UserID.Hex()] = row.ProjectID.Hex()
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return result, nil
}

// GetFeaturedBoardStats retrieves per-member aggregated metrics using a $facet pipeline.
// projectIDs maps each member's UserID to their earliest registered ProjectID.
// memberIDs is the list of user IDs to include.
// since filters activity to only include records at or after this time.
func (r *MongoFeaturedBoardRepository) GetFeaturedBoardStats(
	ctx context.Context,
	projectIDs map[string]string,
	memberIDs []string,
	since time.Time,
) ([]*repository.FeaturedMemberStats, error) {
	if len(projectIDs) == 0 || len(memberIDs) == 0 {
		return []*repository.FeaturedMemberStats{}, nil
	}

	// Build the set of project ObjectIDs to filter sessions
	projectOIDSet := make([]bson.ObjectID, 0, len(projectIDs))
	for _, projectID := range projectIDs {
		oid, err := bson.ObjectIDFromHex(projectID)
		if err != nil {
			r.logger.Warn("invalid project ID, skipping",
				slog.String("projectID", projectID),
				slog.Any("error", err),
			)
			continue
		}
		projectOIDSet = append(projectOIDSet, oid)
	}

	// Build the set of member ObjectIDs to filter sessions
	memberOIDSet := make([]bson.ObjectID, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		oid, err := bson.ObjectIDFromHex(memberID)
		if err != nil {
			r.logger.Warn("invalid member ID, skipping",
				slog.String("memberID", memberID),
				slog.Any("error", err),
			)
			continue
		}
		memberOIDSet = append(memberOIDSet, oid)
	}

	if len(projectOIDSet) == 0 || len(memberOIDSet) == 0 {
		return []*repository.FeaturedMemberStats{}, nil
	}

	// Base match filter: sessions in the specified projects, by specified members, since the given time
	baseMatch := bson.M{
		mongoschema.SessionProjectIDField: bson.M{"$in": projectOIDSet},
		mongoschema.SessionUserIDField:    bson.M{"$in": memberOIDSet},
		mongoschema.SessionTimestampField: bson.M{"$gte": since},
	}

	// $facet pipeline for parallel aggregations
	pipeline := bson.A{
		bson.M{"$match": baseMatch},
		bson.M{"$facet": bson.M{
			// Prompt counts: match human type, group by userId
			"prompts": bson.A{
				bson.M{"$match": bson.M{mongoschema.SessionTypeField: string(session.SessionTypeHuman)}},
				bson.M{"$group": bson.M{
					"_id":                      "$" + mongoschema.SessionUserIDField,
					aggFeaturedPromptCountField: bson.M{"$sum": 1},
				}},
			},
			// Token usage: match agent type, group by userId
			"tokens": bson.A{
				bson.M{"$match": bson.M{mongoschema.SessionTypeField: string(session.SessionTypeAgent)}},
				bson.M{"$group": bson.M{
					"_id":                         "$" + mongoschema.SessionUserIDField,
					aggFeaturedInputTokensField:         bson.M{"$sum": "$" + mongoschema.SessionUsageInputTokensPath},
					aggFeaturedOutputTokensField:        bson.M{"$sum": "$" + mongoschema.SessionUsageOutputTokensPath},
					aggFeaturedCacheCreationTokensField: bson.M{"$sum": "$" + mongoschema.SessionUsageCacheCreationInputTokensPath},
					aggFeaturedCacheReadTokensField:     bson.M{"$sum": "$" + mongoschema.SessionUsageCacheReadInputTokensPath},
				}},
			},
			// Tool calls: match tool_execution type, group by {userId, toolName}
			"tools": bson.A{
				bson.M{"$match": bson.M{mongoschema.SessionTypeField: string(session.SessionTypeToolExecution)}},
				bson.M{"$group": bson.M{
					"_id": bson.M{
						"userId":   "$" + mongoschema.SessionUserIDField,
						"toolName": "$" + mongoschema.SessionToolNameField,
					},
					aggFeaturedToolCountField: bson.M{"$sum": 1},
				}},
			},
			// Last activity: group by userId, max timestamp
			"activity": bson.A{
				bson.M{"$group": bson.M{
					"_id":                       "$" + mongoschema.SessionUserIDField,
					aggFeaturedLastActivityField: bson.M{"$max": "$" + mongoschema.SessionTimestampField},
				}},
			},
		}},
	}

	cursor, err := r.sessionsColl.Aggregate(ctx, pipeline)
	if err != nil {
		r.logger.Error("failed to aggregate featured board stats", slog.Any("error", err))
		return nil, fmt.Errorf("failed to aggregate featured board stats: %w", err)
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		if err := cursor.Err(); err != nil {
			return nil, fmt.Errorf("cursor error: %w", err)
		}
		return []*repository.FeaturedMemberStats{}, nil
	}

	var facetResult struct {
		Prompts []struct {
			ID          bson.ObjectID `bson:"_id"`
			PromptCount int32         `bson:"promptCount"`
		} `bson:"prompts"`
		Tokens []struct {
			ID                  bson.ObjectID `bson:"_id"`
			InputTokens         int64         `bson:"inputTokens"`
			OutputTokens        int64         `bson:"outputTokens"`
			CacheCreationTokens int64         `bson:"cacheCreationTokens"`
			CacheReadTokens     int64         `bson:"cacheReadTokens"`
		} `bson:"tokens"`
		Tools []struct {
			ID struct {
				UserID   bson.ObjectID `bson:"userId"`
				ToolName string        `bson:"toolName"`
			} `bson:"_id"`
			Count int32 `bson:"count"`
		} `bson:"tools"`
		Activity []struct {
			ID             bson.ObjectID `bson:"_id"`
			LastActivityAt time.Time     `bson:"lastActivityAt"`
		} `bson:"activity"`
	}

	if err := cursor.Decode(&facetResult); err != nil {
		return nil, fmt.Errorf("failed to decode facet result: %w", err)
	}

	// Build lookup maps keyed by user hex ID
	promptMap := make(map[string]int32, len(facetResult.Prompts))
	for _, p := range facetResult.Prompts {
		promptMap[p.ID.Hex()] = p.PromptCount
	}

	type tokenEntry struct {
		InputTokens         int64
		OutputTokens        int64
		CacheCreationTokens int64
		CacheReadTokens     int64
	}
	tokenMap := make(map[string]tokenEntry, len(facetResult.Tokens))
	for _, t := range facetResult.Tokens {
		tokenMap[t.ID.Hex()] = tokenEntry{
			InputTokens:         t.InputTokens,
			OutputTokens:        t.OutputTokens,
			CacheCreationTokens: t.CacheCreationTokens,
			CacheReadTokens:     t.CacheReadTokens,
		}
	}

	toolMap := make(map[string]map[string]int32)
	for _, t := range facetResult.Tools {
		userHex := t.ID.UserID.Hex()
		if toolMap[userHex] == nil {
			toolMap[userHex] = make(map[string]int32)
		}
		toolMap[userHex][t.ID.ToolName] += t.Count
	}

	activityMap := make(map[string]time.Time, len(facetResult.Activity))
	for _, a := range facetResult.Activity {
		activityMap[a.ID.Hex()] = a.LastActivityAt
	}

	// Assemble results for each member
	results := make([]*repository.FeaturedMemberStats, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		projectID := projectIDs[memberID]

		tokens := tokenMap[memberID]
		stats := &repository.FeaturedMemberStats{
			UserID:      memberID,
			ProjectID:   projectID,
			PromptCount: promptMap[memberID],
			Usage: repository.TokenUsageSummary{
				TotalInputTokens:         tokens.InputTokens,
				TotalOutputTokens:        tokens.OutputTokens,
				TotalCacheCreationTokens: tokens.CacheCreationTokens,
				TotalCacheReadTokens:     tokens.CacheReadTokens,
			},
			ToolCalls:      toolMap[memberID],
			LastActivityAt: activityMap[memberID],
		}
		results = append(results, stats)
	}

	return results, nil
}

// GetProjectNames returns a map of ProjectID -> ProjectName for the given project IDs.
func (r *MongoFeaturedBoardRepository) GetProjectNames(
	ctx context.Context,
	projectIDs []string,
) (map[string]string, error) {
	if len(projectIDs) == 0 {
		return map[string]string{}, nil
	}

	oids := make([]bson.ObjectID, 0, len(projectIDs))
	for _, id := range projectIDs {
		oid, err := bson.ObjectIDFromHex(id)
		if err != nil {
			r.logger.Warn("invalid project ID, skipping", slog.String("projectID", id), slog.Any("error", err))
			continue
		}
		oids = append(oids, oid)
	}

	if len(oids) == 0 {
		return map[string]string{}, nil
	}

	filter := bson.M{mongoschema.ProjectIDField: bson.M{"$in": oids}}
	cursor, err := r.projectsColl.Find(ctx, filter, nil)
	if err != nil {
		r.logger.Error("failed to get project names", slog.Any("error", err))
		return nil, fmt.Errorf("failed to get project names: %w", err)
	}
	defer cursor.Close(ctx)

	result := make(map[string]string, len(projectIDs))
	for cursor.Next(ctx) {
		var row struct {
			ID   bson.ObjectID `bson:"_id"`
			Name string        `bson:"name"`
		}
		if err := cursor.Decode(&row); err != nil {
			r.logger.Error("failed to decode project name", slog.Any("error", err))
			return nil, fmt.Errorf("failed to decode project name: %w", err)
		}
		result[row.ID.Hex()] = row.Name
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return result, nil
}

// Interface verification
var _ repository.FeaturedBoardRepositoryPort = (*MongoFeaturedBoardRepository)(nil)
