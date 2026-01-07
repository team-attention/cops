---
name: verify-skill
description: |
  Verifies skill documentation for consistency, accuracy, and completeness.
  Checks parameter names, dependent skill interfaces, referenced paths, and examples.
---

# Verify-Skill

Verifies skill documentation for consistency and accuracy. This skill analyzes SKILL.md files and their supporting documents to find contradictions, interface mismatches, and missing references.

## Parameters

### Required

- First argument: Name of the skill to verify (e.g., `plan`, `linear`, `clarify`)

### Optional

- `FIX=true` - Include specific fix suggestions in the report. Omit for report only.

## Usage Examples

```bash
# Verify a single skill
skill: verify-skill
args: plan

# Verify with fix suggestions
skill: verify-skill
args: plan FIX=true
```

## Process

### 1. Locate Skill Files

```
SKILL_DIR=.agent/skills/$SKILL_NAME
```

Verify the skill directory exists. If not, report error and exit.

### 2. Structure Check

Verify required and referenced files exist:

| Check                  | Description                                   |
| :--------------------- | :-------------------------------------------- |
| `SKILL.md` exists      | Main skill definition file must exist         |
| Valid YAML frontmatter | Must have `name` and `description` fields     |
| Scripts exist          | All files referenced in `scripts/` must exist |
| Docs exist             | All files referenced in `docs/` must exist    |
| Links resolve          | All markdown links must point to valid files  |

**How to Verify:**

```bash
test -f .agent/skills/$SKILL_NAME/SKILL.md
ls .agent/skills/$SKILL_NAME/scripts/*.sh 2>/dev/null
ls .agent/skills/$SKILL_NAME/docs/*.md 2>/dev/null
```

### 3. Description Quality Check

Verify the frontmatter description accurately represents the skill's behavior:

| Check                      | Description                                                                     |
| :------------------------- | :------------------------------------------------------------------------------ |
| Accurate behavior coverage | Description should accurately represent what the skill does                     |
| Invocation clarity         | Description should be detailed enough for LLM to know when to invoke this skill |

**How to Verify:**
1. Read the full SKILL.md content (process steps, parameters, examples)
2. Compare with the frontmatter description
3. Check: Does the description cover all key behaviors?
4. Check: Would an LLM know when to use this skill based on the description alone?
5. Flag vague or incomplete descriptions as Medium severity

**Examples of issues:**

| Issue               | Example                                                                         |
| :------------------ | :------------------------------------------------------------------------------ |
| Incomplete coverage | Description says "manages documents" but skill only creates (not update/delete) |
| Unexplained feature | Description mentions "caching" but doesn't explain what is cached               |
| Too generic         | "Linear integration" (could mean anything)                                      |

### 4. Internal Consistency Check

Verify parameter names are consistent across all sections:

| Location           | Should Match        |
| :----------------- | :------------------ |
| Parameters section | Source of truth     |
| Process section    | Must use same names |
| Usage Examples     | Must use same names |
| Output Format      | Must use same names |

**Common Issues:**

| Issue         | Example                                                   |
| :------------ | :-------------------------------------------------------- |
| Name mismatch | Parameters: `TASK_PATH`, Process: `REQUIREMENTS_PATH`     |
| Typo          | Parameters: `ARTIFACT_DIR_PATH`, Example: `ARTIFACT_PATH` |
| Case mismatch | Parameters: `issueId`, Process: `issue_id`                |

**How to Verify:**
1. Extract all parameter names from "Parameters" section
2. Search entire document for each parameter
3. Flag any variations or mismatches

### 5. Dependent Skill Interface Check

Verify all skill references are valid:

**Valid Syntax:**
```
skill: {skill_name}
args: {command} {parameters}
```

**Invalid Patterns to Flag:**

| Pattern                     | Issue                                  |
| :-------------------------- | :------------------------------------- |
| `skill: linear-issue`       | Non-existent skill name                |
| `$(Skill tool: X, args: Y)` | Non-standard syntax                    |
| `skill: artifact args: ...` | Missing newline between skill and args |

**How to Verify:**
1. Find all `skill: {name}` patterns
2. Check `.agent/skills/{name}/SKILL.md` exists
3. Verify command names and parameters match the skill's interface

### 6. Example Accuracy Check

Verify examples match actual behavior:

**Script Output Format:**
1. Run the script (if safe)
2. Compare output with documented format
3. Flag discrepancies

**Example Issue:**
- Documented: `.agent/tmp/prefix-xxxxxx` (using `-`)
- Actual: `.agent/tmp/prefix.xxxxxx` (using `.`)

**Invocation Syntax:**
```
# Correct
skill: mktemp
args: plan

# Incorrect (missing args line)
skill: mktemp plan
```

### 7. Documentation Duplication Check

Check for content overlap between SKILL.md and reference documents (docs/*.md).

**Why This Matters:**
Duplicated content creates maintenance burden - when one copy is updated, the other becomes stale.

**What to Check:**

| Pattern                   | Issue                                               |
| :------------------------ | :-------------------------------------------------- |
| Same tables in both files | Content should live in one place only               |
| Same code examples        | Consolidate or reference, don't duplicate           |
| Same process descriptions | SKILL.md should be self-contained or reference docs |

**How to Verify:**
1. List all `docs/*.md` files in the skill directory
2. For each doc, compare section headings and content with SKILL.md
3. Flag significant overlap (>50% similar content) as Medium severity

**Resolution Options:**

| Option                          | When to Use                                                          |
| :------------------------------ | :------------------------------------------------------------------- |
| Merge into SKILL.md             | When docs add little beyond SKILL.md                                 |
| Keep docs, remove from SKILL.md | When docs have rich detail, SKILL.md should just reference           |
| Keep both with clear separation | When docs serve different audience (e.g., API reference vs tutorial) |

### 8. Reverse Dependency Check

Find and verify skills that depend on the target skill.

**Why This Matters:**
When a skill changes (e.g., output path changes from `/tmp/` to `.agent/tmp/`), dependent skills may have hardcoded references that become outdated.

**How to Find Dependent Skills:**

```bash
grep -r "skill:\s*$SKILL_NAME" .agent/skills/ --include="*.md"
```

**What to Check:**

| Check                  | Description                                          |
| :--------------------- | :--------------------------------------------------- |
| Output path references | Hardcoded paths that match the skill's output format |
| Interface usage        | Commands and parameters still valid                  |
| Example accuracy       | Examples use current output format                   |

**Common Issues:**

| Issue              | Example                                                    |
| :----------------- | :--------------------------------------------------------- |
| Outdated path      | Dependent uses `/tmp/` but skill now outputs `.agent/tmp/` |
| Deprecated command | Dependent uses old command name                            |
| Changed parameter  | Dependent passes parameter that no longer exists           |

**How to Verify:**
1. Search for `skill: {SKILL_NAME}` in all skill directories
2. For each dependent skill, extract hardcoded paths from examples
3. Compare with target skill's documented output format
4. Flag mismatches as High severity

### 9. Generate Report

Output a verification report:

```markdown
# Skill Verification Report: {SKILL_NAME}

## Summary
- Total Issues: N
- Critical: X
- High: Y
- Medium: Z

## Issues

### Critical
{Issues that will cause failures}

### High
{Issues that cause confusion or maintenance problems}

### Medium
{Documentation quality issues}

## Suggested Fixes (if FIX=true)
{Specific edit suggestions with file:line references}
```

## Severity Definitions

| Severity | Criteria                        | Examples                                                                     |
| :------- | :------------------------------ | :--------------------------------------------------------------------------- |
| Critical | Skill will fail to execute      | Non-existent skill reference, wrong command name, missing required parameter |
| High     | Confusion or incorrect behavior | Parameter name mismatch, outdated path references, example output mismatch   |
| Medium   | Documentation quality issue     | Missing error handling docs, incomplete descriptions, missing examples       |

## Output

Verification report as markdown, listing all found issues by severity.
