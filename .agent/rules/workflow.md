---
trigger: always_on
---

# Workflow Rules

## Pre-Action Context Loading

Before starting any task, planning, or answering questions about specific parts of the codebase:

### 1. Identify Target Directory

Use available context (git status, file paths mentioned, project structure) to determine which module/directory will be affected.

**Don't guess based on task keywords alone** - Use actual project information:
- Check recent git changes if mentioned
- Look at file paths in the conversation
- Use `ls` or directory structure from loaded context

### 2. Load Directory Context & Rules by Reading Example Files

**If similar implementation files exist:**
- Read **example implementation files** from:
  - The same package where you'll be working
  - Or similar packages at the same level (e.g., if working in `api/internal/handler/`, read examples from other handlers)

- Reading example files serves two purposes:
  1. Automatically loads applicable rules from `.agent/rules/`
  2. Shows actual implementation patterns to follow

**If example code conflicts with rules:**
- **Prioritize rules over existing code**
- Inform the user: "I noticed the existing code at `{file}:{line}` doesn't follow the rule in `.agent/rules/{rule-file}`. I'll implement this following the rule instead."
- Follow the rule from `.agent/rules/` when implementing new code

**If no similar implementation files exist (new package or no comparable examples):**
- Explicitly read applicable rule files from `.agent/rules/`
- Use `Glob` to discover what rules exist for the language/framework you're working with

### 3. Apply Loaded Rules

After loading context and rules:
- Create todos that follow the loaded rules
- Make architectural decisions based on project patterns (if they align with rules)

## Key Principles

1. **Always load actual code/rules before planning** - Don't rely on assumptions
2. **Read example files to understand patterns** - Learn from existing implementation
3. **Rules override code when they conflict** - Rules are the source of truth
4. **Inform user about conflicts** - Transparency about implementation decisions
