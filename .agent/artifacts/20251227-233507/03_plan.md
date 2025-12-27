# Implementation Plan

## Overview
Add an `onClick` handler to `TableRow` in `ProjectsTable` component that navigates to the project detail page, making the entire row clickable while maintaining the existing `<Link>` in the first cell for accessibility.

## Selected Packages

| Problem    | Package              | Context7 ID        | Reason for Selection                                                                 |
| ---------- | -------------------- | ------------------ | ------------------------------------------------------------------------------------ |
| Navigation | @tanstack/react-router | `/tanstack/router` | Already in use in the project; provides `useNavigate` hook for programmatic navigation |

## Architecture Decisions

### Decision 1: Use `useNavigate` hook for row click navigation
**Choice**: Use TanStack Router's `useNavigate` hook to programmatically navigate when the row is clicked
**Rationale**: This is the established pattern in the codebase (seen in `/Users/jayce/team-attention/cops/web/src/route/projects/index.tsx` and `/Users/jayce/team-attention/cops/web/src/route/sessions/index.tsx`). It allows navigation from any click target while keeping the `<Link>` component for accessibility.

### Decision 2: Keep existing `<Link>` in first cell
**Choice**: Maintain the `<Link>` component in the first `TableCell` alongside the new row-level `onClick`
**Rationale**: The `<Link>` element is required for:
- Keyboard navigation (Tab to focus, Enter to activate)
- Screen reader accessibility (announces as a link)
- Browser features (right-click to open in new tab, Ctrl+click)

### Decision 3: No event propagation handling needed
**Choice**: Do not add `stopPropagation()` or special click handling for tooltips
**Rationale**: Per requirements, clicking anywhere on the row (including tooltip triggers) should navigate. The tooltip displays on hover, not click, so there's no conflict.

## Implementation Steps

### Step 1: Modify ProjectsTable component

**Files to Modify**:
- `/Users/jayce/team-attention/cops/web/src/feature/project/component/projects-table.tsx` (modify)

**Changes**:

#### Change 1.1: Update import statement (Line 1)

Add `useNavigate` to the existing `@tanstack/react-router` import.

**Current code (line 1)**:
```tsx
import { Link } from '@tanstack/react-router'
```

**New code**:
```tsx
import { Link, useNavigate } from '@tanstack/react-router'
```

#### Change 1.2: Add navigate hook inside component (Line 54, after function declaration)

Add the `useNavigate` hook call at the beginning of the component function, before the early return for empty projects.

**Current code (lines 53-54)**:
```tsx
export const ProjectsTable = ({ projects, sortBy, sortDesc, onSortChange }: ProjectsTableProps) => {
  if (projects.length === 0) {
```

**New code**:
```tsx
export const ProjectsTable = ({ projects, sortBy, sortDesc, onSortChange }: ProjectsTableProps) => {
  const navigate = useNavigate()

  if (projects.length === 0) {
```

#### Change 1.3: Add onClick handler to TableRow (Lines 91-93)

Add an `onClick` handler to the `TableRow` component that navigates to the project detail page.

**Current code (lines 91-93)**:
```tsx
            <TableRow
              key={project.id}
              className="group cursor-pointer border-zinc-800/30 transition-colors hover:bg-zinc-800/30"
            >
```

**New code**:
```tsx
            <TableRow
              key={project.id}
              className="group cursor-pointer border-zinc-800/30 transition-colors hover:bg-zinc-800/30"
              onClick={() => navigate({ to: '/projects/$projectId', params: { projectId: project.id } })}
            >
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
| -------- | ----- | --------------- | -------------- |
| Click on project name (first cell) | Mouse click on project name link | Navigates to `/projects/{id}` | Both Link and onClick handler |
| Click on path column (second cell) | Mouse click on path text | Navigates to `/projects/{id}` | onClick handler only |
| Click on sessions count (third cell) | Mouse click on sessions number | Navigates to `/projects/{id}` | onClick handler only |
| Click on tokens column with tooltip | Mouse click on tokens area | Navigates to `/projects/{id}` | onClick handler (tooltip shows on hover) |
| Click on activity column (last cell) | Mouse click on activity time | Navigates to `/projects/{id}` | onClick handler only |
| Hover on tokens column | Mouse hover on tokens area | Tooltip displays with token breakdown | Tooltip functionality preserved |
| Keyboard Tab navigation | Press Tab key | Focus moves to Link in first cell | Accessibility - Link focusable |
| Keyboard Enter on focused link | Press Enter when Link is focused | Navigates to `/projects/{id}` | Accessibility - Link activatable |
| Empty projects list | Empty array passed as props | Shows "No projects found" message | Early return branch |
| Right-click on project name | Right-click on link | Browser context menu appears | Link context menu preserved |

## Execution Order

1. Step 1.1: Update import statement (no dependencies)
2. Step 1.2: Add navigate hook (depends on 1.1 - uses `useNavigate`)
3. Step 1.3: Add onClick handler to TableRow (depends on 1.2 - uses `navigate` function)

## Notes for Execute Agent

- All changes are in a single file: `/Users/jayce/team-attention/cops/web/src/feature/project/component/projects-table.tsx`
- The changes are minimal and additive - no existing code is removed or modified except the import line
- The `cursor-pointer` class already exists on `TableRow`, so visual feedback is already correct
- Do NOT modify `SessionsTable` - it is only a reference for the pattern, not a target for changes
- Verify the component still renders correctly after changes by running the dev server
- Test both mouse click navigation and keyboard navigation (Tab + Enter) to ensure accessibility
- Test tooltip hover functionality to ensure it still works (hover shows tooltip, click navigates)
