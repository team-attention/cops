# Implementation Plan: Fix Organization Selection Step Not Appearing

## Overview

This plan addresses a user-reported issue where the organization selection step in the TUI is being skipped. The root cause is auto-skip logic in `add_tui_update.go` that bypasses the organization selection UI when a user has only one organization. This implementation removes that auto-skip behavior to ensure users always see and confirm their organization selection, regardless of how many organizations they have.

## Package Changes

None required. All changes use existing packages.

## Implementation Steps

### Step 1: Remove Auto-Skip Logic in orgFetchMsg Handler

**Files to Read**:
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-backend.md`: General Go coding standards
- `/Users/jayce/team-attention/cops/.agent/rules/go/go-logging-conventions.md`: Logging standards for adding debug logging
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go`: Current implementation to modify

#### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go`

**Description**:
Remove the auto-skip logic (lines 39-46) that automatically selects a single organization and skips to git selection. The organization selection UI must always be displayed.

**Current Code to Remove** (lines 39-46):
```go
if len(msg.organizations) == 1 {
    // Auto-select single organization
    m.selectedOrgID = string(msg.organizations[0].ID)
    m.selectedOrgName = msg.organizations[0].Name
    m.result.OrganizationID = m.selectedOrgID
    m.step = stepGitSelection
    return m, m.detectGitRepos
}
```

**Modified orgFetchMsg Case** (replace lines 29-48):
```go
case orgFetchMsg:
	// 1. Handle error from organization fetch
	if msg.err != nil {
		m.err = msg.err
		return m, tea.Quit
	}
	// 2. Store fetched organizations in model
	m.organizations = msg.organizations
	// 3. Handle edge case: no organizations found
	if len(msg.organizations) == 0 {
		m.err = fmt.Errorf("no organizations found. Please create an organization first")
		return m, tea.Quit
	}
	// 4. Always show organization selection UI (removed auto-skip for single org)
	//    User must explicitly select/confirm their organization
	return m, nil
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Single organization | `orgFetchMsg{organizations: [1 org], err: nil}` | Stay on stepOrgSelection, organizations populated | Single org branch (previously auto-skipped) |
| Multiple organizations | `orgFetchMsg{organizations: [3 orgs], err: nil}` | Stay on stepOrgSelection, organizations populated | Multiple org branch |
| No organizations | `orgFetchMsg{organizations: [], err: nil}` | Error set, tea.Quit returned | Zero org error branch |
| Fetch error | `orgFetchMsg{organizations: nil, err: someError}` | Error set, tea.Quit returned | Error handling branch |

---

### Step 2: Update View Text for Single Organization Case

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go`: Current view implementation

#### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go`

**Description**:
Update the `viewOrgSelection` function to display a contextual message based on whether the user has one or multiple organizations. For single-org users, show "Confirm organization" instead of "Choose which organization".

**Modified viewOrgSelection Function** (replace lines 33-59):
```go
// viewOrgSelection renders the organization selection view.
func (m addModel) viewOrgSelection() string {
	var b strings.Builder

	// 1. Show loading state when organizations not yet fetched
	if len(m.organizations) == 0 {
		b.WriteString("Fetching organizations...\n")
		return b.String()
	}

	// 2. Render title
	b.WriteString(m.titleStyle.Render("Select Organization"))
	b.WriteString("\n\n")

	// 3. Show contextual instruction based on organization count
	if len(m.organizations) == 1 {
		// Single org: user confirms rather than chooses
		b.WriteString("Confirm organization for this project:\n\n")
	} else {
		// Multiple orgs: user chooses from list
		b.WriteString("Choose which organization to add this project to:\n\n")
	}

	// 4. Render organization list with cursor
	for i, org := range m.organizations {
		cursor := "  "
		if m.orgCursor == i {
			cursor = m.cursorStyle.Render("> ")
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, org.Name))
	}

	// 5. Render help text
	b.WriteString("\n")
	b.WriteString(m.helpStyle.Render("up/down: navigate | enter: select | ctrl+c: cancel"))
	b.WriteString("\n")

	return b.String()
}
```

**Test Scenarios**:

| Scenario | Input State | Expected Output | Branch Covered |
|:---------|:------------|:----------------|:---------------|
| Loading state | `m.organizations = nil` | "Fetching organizations..." | Loading branch |
| Single organization | `m.organizations = [1 org]` | Title + "Confirm organization for this project:" + org list | Single org message branch |
| Multiple organizations | `m.organizations = [3 orgs]` | Title + "Choose which organization to add this project to:" + org list | Multiple org message branch |
| Cursor on first org | `m.orgCursor = 0` | First org has "> " prefix | Cursor rendering |
| Cursor on second org | `m.orgCursor = 1` | Second org has "> " prefix | Cursor rendering |

---

### Step 3: Add OrganizationID Validation in add.go

**Files to Read**:
- `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go`: Current command implementation

#### `/Users/jayce/team-attention/cops/cli/internal/service/tracking/inbound/cli/cobra/add.go`

**Description**:
Add validation after the cancellation check to ensure `OrganizationID` is not empty before proceeding with project registration. This prevents projects from being created without an organization association.

**Modified RunE Function** (insert validation after line 56, before line 58):
```go
// Check if user cancelled
if result.Cancelled {
	fmt.Println("Operation cancelled.")
	return nil
}

// Validate organization was selected
// 1. OrganizationID is required for all projects
// 2. If empty, the TUI flow was interrupted or has a bug
// 3. Return user-friendly error message
if result.OrganizationID == "" {
	return fmt.Errorf("organization selection is required. Please try again")
}

// Create params from TUI result
params := tracking.AddProjectParams{
```

**Test Scenarios**:

| Scenario | Input | Expected Output | Branch Covered |
|:---------|:------|:----------------|:---------------|
| Valid OrganizationID | `result.OrganizationID = "org123"` | Continue to AddProject | Happy path |
| Empty OrganizationID | `result.OrganizationID = ""` | Error: "organization selection is required" | Validation error branch |
| Cancelled operation | `result.Cancelled = true` | "Operation cancelled.", return nil | Cancellation branch (unchanged) |

---

## Summary of Changes

| File | Change Type | Description |
|:-----|:------------|:------------|
| `cli/internal/service/tracking/inbound/cli/cobra/add_tui_update.go` | Modify | Remove auto-skip logic (lines 39-46) for single organization case |
| `cli/internal/service/tracking/inbound/cli/cobra/add_tui_view.go` | Modify | Update `viewOrgSelection` to show contextual message for single vs multiple orgs |
| `cli/internal/service/tracking/inbound/cli/cobra/add.go` | Modify | Add OrganizationID validation after cancellation check |

## Verification Checklist

After implementation, verify:

1. **Single Organization User**:
   - Run `cops add .`
   - Organization selection UI MUST appear showing "Confirm organization for this project:"
   - User can press Enter to confirm
   - Project is created with OrganizationID

2. **Multiple Organization User**:
   - Run `cops add .`
   - Organization selection UI MUST appear showing "Choose which organization to add this project to:"
   - User can navigate and select
   - Project is created with selected OrganizationID

3. **No Organizations**:
   - User has 0 organizations
   - Error message: "no organizations found. Please create an organization first"
   - TUI exits gracefully

4. **Validation**:
   - If OrganizationID is somehow empty after TUI completes (edge case)
   - Error message: "organization selection is required. Please try again"
   - Project is NOT created
