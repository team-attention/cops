# Requirements

## Request Summary
Make the entire `TableRow` in `ProjectsTable` clickable to navigate to the project detail page. Currently, only the project name (first column) is clickable, creating a UX mismatch where the row appears fully clickable (cursor-pointer, hover effects) but only responds to clicks on the name. This should follow the same implementation pattern already established in `SessionsTable`.

## Acceptance Criteria

- [ ] Entire `TableRow` is clickable and navigates to `/projects/$projectId` on click
- [ ] Navigation occurs immediately on row click, regardless of which cell is clicked
- [ ] Tooltip in the Tokens column still displays on hover (clicking the tooltip also navigates)
- [ ] Keyboard accessibility is maintained: Tab navigation focuses the link in the first cell
- [ ] Screen reader accessibility is maintained: Link element exists for assistive technologies
- [ ] Implementation follows the exact pattern used in `SessionsTable` (Link in first cell + onClick on TableRow)
- [ ] All existing visual feedback remains unchanged (hover background, color transitions, group-hover effects)
- [ ] No console errors or warnings in the browser

## Scope

### In Scope
- Modify `ProjectsTable` component in `/Users/jayce/team-attention/cops/web/src/feature/project/component/projects-table.tsx`
- Add `onClick` handler to `TableRow` component for navigation
- Maintain existing `<Link>` component in first `TableCell` for accessibility
- Use TanStack Router's `useNavigate` hook for programmatic navigation in onClick handler

### Out of Scope
- Modifying visual styling or hover effects (keep exactly as-is)
- Changes to sorting functionality
- Changes to data fetching or state management
- Modifications to other table components
- Adding new interactive elements or tooltips

## Constraints
- Must use TanStack Router for navigation (already in use: `@tanstack/react-router`)
- Must maintain existing accessibility features (keyboard navigation, screen readers)
- Implementation must follow established pattern in `SessionsTable` for consistency
- No breaking changes to existing props or component API
- No changes to existing visual design (styling must remain identical)

## Additional Context

### Reference Implementation
The `SessionsTable` component at `/Users/jayce/team-attention/cops/web/src/feature/session/component/sessions-table.tsx` demonstrates the desired pattern:
- Lines 105-108: `TableRow` with cursor-pointer and hover styles
- Lines 109-131: First `TableCell` contains `<Link>` component for accessibility
- The entire row should respond to clicks while maintaining keyboard navigation

### Current Implementation Analysis
File: `/Users/jayce/team-attention/cops/web/src/feature/project/component/projects-table.tsx`
- Lines 91-93: `TableRow` already has cursor-pointer and hover styling
- Lines 95-105: Only first `TableCell` contains navigable `<Link>`
- Lines 106-151: Other cells are not clickable, creating UX inconsistency

### Technical Approach
1. Import `useNavigate` from `@tanstack/react-router`
2. Get navigate function inside component
3. Add `onClick` handler to `TableRow` that calls `navigate({ to: '/projects/$projectId', params: { projectId: project.id } })`
4. Keep existing `<Link>` in first cell for accessibility

## Questions Resolved

| Question | Answer |
| -------- | ------ |
| Should navigation occur on row click even when clicking interactive elements like tooltips? | Yes - entire row navigates immediately on click, tooltips work on hover but clicking also navigates |
| What accessibility pattern should be used for clickable rows? | Keep `<Link>` in first cell for keyboard/screen reader support + add `onClick` to `TableRow` for mouse clicks (same as `SessionsTable`) |
| Should visual feedback or styling be modified? | No - keep all current styling exactly as-is |
| Should this follow an existing pattern in the codebase? | Yes - follow the `SessionsTable` implementation pattern for consistency |
