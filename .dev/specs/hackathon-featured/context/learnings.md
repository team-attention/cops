## TODO 1
- buf binary is available at web/node_modules/@bufbuild/buf/bin/buf (not in system PATH); also installable via homebrew
- The project uses Req/Res suffix convention for RPC message names (not Request/Response), consistent across all proto files
- FeaturedBoardService generates a separate TS connect-query file: dashboard-FeaturedBoardService_connectquery.ts
- buf generate output paths are relative to idl/protobuf directory: ../../shared/gen/grpcstub and ../../web/src/gen/grpcstub
- buf lint exits non-zero (code 100) due to project-wide pre-existing naming convention violations (Req/Res vs Request/Response) — this is intentional project style

## TODO 2
- Go 1.26 is incompatible with sonic v1.14.2 (GoMapIterator undefined) — pre-existing project issue affecting `go build ./api/...` and `go test ./api/...` for some packages
- FeaturedBoardService uses organization/outbound/repository.OrganizationRepositoryPort (already provided by newOrganizationModule)
- Featured board handler is public (no JWT) — registered as PublicConnectHandler with group:public_connect_handlers
- Sessions collection has both userId and projectId fields allowing per-user stats aggregation
- $facet aggregation enables parallel execution of multiple aggregation sub-pipelines in single MongoDB query
- No unit tests were written for FeaturedBoardService (accepted — no test requirement in acceptance criteria)

## TODO 3
- TokenUsageSummary fields (totalInputTokens, totalOutputTokens, etc.) are non-optional bigint — no ?? 0n fallback needed
- TanStack Router validateSearch uses plain function pattern (Record → T), not zod validators (zod is not a dependency)
- Animation keyframes for inactive cards defined in route component via inline `<style>` tag to avoid modifying global CSS
- npm run test exits with code 1 because no test files exist — pre-existing baseline condition
- isFeaturedRoute follows same pattern as isAuthRoute, isOrganizationNewRoute, isInviteRoute in __root.tsx
- npm run check and npm run lint both report 22 pre-existing errors (baseline), none from featured files
