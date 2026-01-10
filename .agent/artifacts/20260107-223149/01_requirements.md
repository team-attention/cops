# Requirements: TA-138 Hook Configuration Structure Design

## Task Summary

Design and implement the configuration file structure for sending Claude Code Hook events to the API server. Project-specific Hook definitions are stored in `.claude/settings.json` (Git-tracked), while API authentication keys are stored in `~/.cops/auth.json` (user-specific, Git-ignored).

## Acceptance Criteria

- [ ] `.claude/settings.json` Hook configuration schema definition
- [ ] `~/.cops/auth.json` API key schema definition
- [ ] Configuration file load logic implementation (Project + Global merge)
- [ ] Hook event type enable/disable settings support
- [ ] Error handling logic for missing API key
- [ ] Configuration file validation logic implementation
- [ ] Unit test creation

## Scope

### In Scope

* **Go struct definitions**: Define `HookConfig`, `AuthConfig` structures in `shared/config/` package
* **JSON parsing implementation**: Implement functions to read `.claude/settings.json` and `~/.cops/auth.json` files and convert to Go structs
* **Configuration loader implementation**: Implement `LoadConfig(projectDir string) (*Config, error)` function (Project + Global config merge)
* **Event filter implementation**: Implement `IsEventEnabled(eventType string) bool` method
* **Error handling**: Handle cases for file not found, JSON parsing failure, missing required fields, etc.

### Out of Scope

* API server handler implementation (separate task)
* Protobuf message definitions (handled in TA-137)
* API key issuance/revocation web UI (handled in TA-139)
* Hook event actual transmission logic (separate task)

## Constraints

* **File locations**:
  * Project: `.claude/settings.json` (Git tracked)
  * Global: `~/.cops/auth.json` (user-specific, Git ignored)
* **Protocol**: Uses `EventService.SendEvents` RPC defined in TA-137
* **Authentication**: `Authorization: Bearer {api_key}` header (see TA-139)
* **Event types**: PostToolUse, Notification, UserPromptSubmit, Stop, SubagentStop, SessionStart, SessionEnd

## Related Tasks

* **TA-137**: Hook event protocol design (protobuf message definitions)
* **TA-139**: API key authentication design (issuance/verification methods)

## Source

Linear Issue: [TA-138](https://linear.app/team-attention/issue/TA-138/hook-%EC%84%A4%EC%A0%95-%EA%B5%AC%EC%A1%B0-%EC%84%A4%EA%B3%84)
