# Research Report

## Mode
General Research

## Request Summary
Make the entire `TableRow` in `ProjectsTable` clickable to navigate to the project detail page. Currently, only the project name (first column) is clickable via a `<Link>` component. The row visually appears clickable (cursor-pointer, hover effects) but only responds to clicks on the name. This should follow the pattern already established in `SessionsTable`.

## Files to Read Before Planning

Before creating the implementation plan, the Planning Agent MUST read these files:

| File | Reason |
| ---- | ------ |
| `/Users/jayce/team-attention/cops/web/src/feature/session/component/sessions-table.tsx` | Reference implementation - shows how clickable rows should work with Link in first cell |
| `/Users/jayce/team-attention/cops/web/src/feature/project/component/projects-table.tsx` | Target file to modify - current implementation that needs row click handler |
| `/Users/jayce/team-attention/cops/.agent/rules/react/react-web.md` | React/TypeScript rules for the project |
| `/Users/jayce/team-attention/cops/.agent/rules/react/react-web-src.md` | Feature organization rules and TanStack Router patterns |

## Package Candidates

### Problem 1: Navigation

No external packages needed. TanStack Router is already installed and in use.

| Package | Context7 ID | Why Better Than Alternatives |
| ------- | ----------- | ---------------------------- |
| @tanstack/react-router | `/tanstack/router` | Already in use in the project. Provides `useNavigate` hook for programmatic navigation. |

## Technical Constraints

1. **Must maintain accessibility** - Keep the existing `<Link>` component in the first cell for keyboard navigation and screen reader support
2. **No visual changes** - All existing styling (cursor-pointer, hover effects, group-hover) must remain unchanged
3. **Follow existing pattern** - Implementation must match the `SessionsTable` approach for codebase consistency
4. **TanStack Router requirement** - Must use the `useNavigate` hook pattern established in the codebase

## Similar Implementations Found

### Example 1: SessionsTable Row Structure (REFERENCE PATTERN)
- **File**: `/Users/jayce/team-attention/cops/web/src/feature/session/component/sessions-table.tsx:105-131`
- **Relevance**: This is the exact pattern to follow. Shows:
  - `TableRow` with `cursor-pointer` and hover styling (line 107)
  - First `TableCell` contains `<Link>` for accessibility (lines 109-131)
  - Other cells are non-interactive but the row appears fully clickable

**Note**: The `SessionsTable` currently does NOT have an `onClick` handler on the `TableRow`. The requirements document indicates that `SessionsTable` should be the reference pattern, but upon inspection, it also only has the `<Link>` in the first cell without row-level click handling. The implementation for `ProjectsTable` should add the `onClick` handler to `TableRow` as specified in the requirements.

### Example 2: Route-level useNavigate Usage
- **File**: `/Users/jayce/team-attention/cops/web/src/route/projects/index.tsx:1,28`
- **Relevance**: Shows how `useNavigate` is imported and used in this codebase:
  ```tsx
  import { createFileRoute, useNavigate, Link } from '@tanstack/react-router'
  // ...
  const navigate = useNavigate({ from: '/projects' })
  ```

### Example 3: Sessions Route Navigation
- **File**: `/Users/jayce/team-attention/cops/web/src/route/sessions/index.tsx:1,31`
- **Relevance**: Another example of `useNavigate` pattern with the `from` option

## TypeScript Types/Interfaces Used

### ProjectSummary (from generated protobuf)
- **File**: `/Users/jayce/team-attention/cops/web/src/gen/grpcstub/dashboard/v1/dashboard_pb.ts:136-178`
- **Key Fields**:
  - `id: string` - Project unique identifier (used for navigation)
  - `name: string` - Project display name
  - `path: string` - Project path
  - `sessionCount: number` - Number of sessions
  - `usage?: TokenUsageSummary` - Aggregated token usage
  - `lastActivity?: Timestamp` - Last activity timestamp

### ProjectsTableProps (inline interface)
- **File**: `/Users/jayce/team-attention/cops/web/src/feature/project/component/projects-table.tsx:17-22`
- **Definition**:
  ```tsx
  interface ProjectsTableProps {
    projects: ProjectSummary[]
    sortBy: SortField
    sortDesc: boolean
    onSortChange: (field: SortField) => void
  }
  ```

## TanStack Router Navigation Patterns

### useNavigate Hook Usage
Based on Context7 documentation and codebase examples:

```tsx
import { useNavigate } from '@tanstack/react-router'

function Component() {
  const navigate = useNavigate()

  const handleClick = () => {
    navigate({ to: '/posts/$postId', params: { postId: '123' } })
  }
}
```

### Link Component Usage (for accessibility)
```tsx
import { Link } from '@tanstack/react-router'

<Link
  to="/projects/$projectId"
  params={{ projectId: project.id }}
  className="block"
>
  {project.name}
</Link>
```

## Implementation Approach

1. **Import `useNavigate`** from `@tanstack/react-router` (add to existing `Link` import)
2. **Create navigate function** inside the component: `const navigate = useNavigate()`
3. **Add `onClick` handler** to each `TableRow`:
   ```tsx
   <TableRow
     key={project.id}
     className="group cursor-pointer border-zinc-800/30 transition-colors hover:bg-zinc-800/30"
     onClick={() => navigate({ to: '/projects/$projectId', params: { projectId: project.id } })}
   >
   ```
4. **Keep existing `<Link>`** in first `TableCell` for accessibility

## Additional Information for Planning

- The `TableRow` component from shadcn/ui accepts standard `<tr>` props including `onClick`
- The route `/projects/$projectId` already exists (see `/Users/jayce/team-attention/cops/web/src/route/projects/$projectId.tsx`)
- No changes needed to the `ProjectsTableProps` interface - the component already has access to `project.id`
- The tooltip in the Tokens column uses `TooltipTrigger` with `asChild` which should not interfere with click propagation
