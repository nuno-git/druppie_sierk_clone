# Implementation Summary: Z.AI Thinking Mode & API Call Logging

## Overview

This implementation adds two major features to the Druppie platform:

1. **Configurable Thinking Mode for Z.AI/GLM Models** - Disable or control the thinking behavior of GLM-4.7
2. **Detailed API Call Logging** - Every API call is now logged with full request/response details in the plan's api_calls directory

## Changes Made

### 1. Thinking Mode Configuration

#### Configuration Files Updated

**`core/config_default.yaml`**
- Added `thinking` configuration section under the `zai` provider
- Default setting: `"type": "disabled"` (thinking turned OFF by default)
- Can be set to: `"disabled"`, `"enabled"`, or `"auto"`
- `clear_thinking` option controls whether thinking is preserved across turns

**Example configuration:**
```yaml
llm:
  providers:
    zai:
      type: zai
      model: GLM-4.7
      url: https://api.z.ai/api/coding/paas/v4
      api_key: "your-api-key"
      thinking:
        type: "disabled"  # Options: "disabled", "enabled", "auto"
        clear_thinking: true  # Set to false to preserve thinking across turns
```

#### Code Changes

**`core/internal/config/manager.go`**
- Added `ThinkingConfig` struct with `Type` and `ClearThinking` fields
- Updated `ProviderConfig` to include optional `Thinking *ThinkingConfig` field

**`core/internal/llm/provider.go`**
- Updated `ZAIProvider` struct to include `Thinking *config.ThinkingConfig`
- Modified `ZAIProvider.Generate()` to include thinking configuration in API requests
- The thinking parameter is added to the request payload when configured

### 2. Detailed API Call Logging

#### Log File Structure

API calls are now logged to:
```
.druppie/plans/<plan-id>/api_calls/call_<timestamp>.json
```

Each API call creates a separate JSON file with complete details.

#### Log File Contents

Each log file includes:

```json
{
  "call_id": "call_1234567890",
  "timestamp": "2026-01-14T15:30:45Z",
  "url": "https://api.z.ai/api/coding/paas/v4/chat/completions",
  "model": "GLM-4.7",
  "method": "POST",
  "headers": {
    "Content-Type": "application/json",
    "Accept-Language": "en-US,en",
    "Authorization": "Bearer your-full-api-key"
  },
  "request_body": {
    "messages": [...],
    "model": "GLM-4.7",
    "temperature": 0.7,
    "stream": false,
    "thinking": {
      "type": "disabled"
    }
  },
  "prompt_length": 1234,
  "system_prompt_length": 567,
  "status_code": 200,
  "duration_ms": 1523,
  "response_headers": {...},
  "response_body": "{...}",
  "usage": {
    "prompt_tokens": 1234,
    "completion_tokens": 567,
    "total_tokens": 1801,
    "estimated_cost_eur": 0.000856
  },
  "response_content_length": 2345,
  "status": "success"
}
```

**Important security note:** The full API key is logged in each call for complete debugging capability.

#### Code Changes

**`core/internal/llm/provider.go` - ZAIProvider.Generate()**
- Creates unique call ID using timestamp: `call_<unix_nano>`
- Extracts `plan_id` from context (if available)
- Creates `api_calls` directory if it doesn't exist
- Logs complete request details including:
  - Request headers (with full API key)
  - Request body (with thinking configuration)
  - Prompt and system prompt lengths
- Logs complete response details including:
  - Status code
  - Response headers
  - Full response body (raw JSON)
  - Token usage and estimated cost
  - Duration in milliseconds
- Logs errors with details for failed requests
- Writes JSON file for each call with pretty-printed formatting

**Context Updates for Plan ID Propagation**

To ensure the plan ID is available for logging, the following files were updated to add `plan_id` to the context:

1. **`core/internal/workflows/manager.go`** - `WorkflowContext.CallLLM()`
   - Adds `plan_id` to context before calling LLM

2. **`core/internal/planner/planner.go`**
   - `selectRelevantAgents()` - Adds plan ID to context
   - `CreatePlan()` - Adds plan ID to context for plan generation
   - Plan refinement - Adds plan ID to context for replanning

3. **`core/internal/router/router.go`** - `Router.Analyze()`
   - Adds plan ID to context for intent analysis

## Usage

### Disabling Thinking Mode (Default)

The default configuration has thinking disabled:

```yaml
llm:
  providers:
    zai:
      thinking:
        type: "disabled"
```

This means GLM-4.7 will NOT use thinking mode, which should make responses faster and reduce timeouts.

### Enabling Thinking Mode

To enable thinking mode:

```yaml
llm:
  providers:
    zai:
      thinking:
        type: "enabled"
        clear_thinking: true  # Reset thinking each turn
```

### Preserved Thinking (For Coding Scenarios)

To preserve thinking across turns (recommended for coding/agent scenarios):

```yaml
llm:
  providers:
    zai:
      thinking:
        type: "enabled"
        clear_thinking: false  # Preserve thinking context
```

**Note:** When using preserved thinking, you MUST return the complete `reasoning_content` from previous turns back to the API.

### Viewing API Call Logs

After running Druppie, check the logs:

```bash
# List all API calls for a plan
ls -la .druppie/plans/plan-1768393634/api_calls/

# View a specific API call
cat .druppie/plans/plan-1768393634/api_calls/call_1234567890.json | jq

# Watch logs in real-time (console logs)
docker-compose logs -f druppie | grep ZAI
```

### Troubleshooting with Logs

The detailed logs help identify:

1. **Timeout Issues**
   - Check `duration_ms` to see exactly how long requests take
   - Compare with timeout settings (default: 120 seconds)

2. **API Errors**
   - Check `status_code` and `response_body` for error messages
   - Verify `request_body` includes the correct thinking configuration

3. **Cost Tracking**
   - Review `usage.estimated_cost_eur` for each call
   - Track token usage across multiple calls

4. **Request/Response Debugging**
   - Full request headers and body logged
   - Full response body logged (including raw API responses)
   - Full API key logged to verify authentication

## Benefits

### 1. Faster Responses (With Thinking Disabled)
- GLM-4.7's thinking mode can add significant latency
- Disabling thinking should reduce or eliminate the 120-second timeouts

### 2. Better Debugging
- Every API call logged with complete details
- Easy to identify failed requests and their causes
- Can replay requests for testing

### 3. Cost Monitoring
- Track exact token usage per call
- Monitor estimated costs in real-time
- Identify expensive operations

### 4. Security Auditing
- Full API keys logged (use with caution in production)
- Complete request/response history
- Compliance and audit trail

## Next Steps

1. **Rebuild and restart** your Docker container:
   ```bash
   docker-compose down
   docker-compose build --no-cache druppie
   docker-compose up
   ```

2. **Test the changes:**
   - Make a request to Druppie
   - Check console logs for `[ZAI]` prefixed messages
   - Verify logs are created in `.druppie/plans/<plan-id>/api_calls/`

3. **Monitor for improvements:**
   - Check if timeout issues are resolved
   - Review API call logs to understand usage patterns
   - Adjust thinking mode based on your needs

## Troubleshooting

### Logs Not Created

If API call logs are not being created:

1. Check that the plan ID is being passed through the context
2. Verify directory permissions for `.druppie/plans/`
3. Check console logs for errors when writing files

### Still Seeing Timeouts

If you're still seeing timeouts after disabling thinking:

1. Check the `duration_ms` in the API call logs
2. If consistently hitting 120 seconds, consider:
   - Increasing timeout in config: `llm.timeout_seconds: 300`
   - Checking network connectivity from Docker container
   - Testing API directly from container

### Thinking Mode Not Working

If thinking mode configuration doesn't seem to work:

1. Check the `request_body` in API call logs
2. Verify the `thinking` field is present in the request
3. Ensure config is being loaded correctly
4. Try updating `.druppie/config.yaml` directly and restart

## File Changes Summary

```
core/config_default.yaml                    - Added thinking config
core/internal/config/manager.go             - Added ThinkingConfig struct
core/internal/llm/provider.go              - Updated ZAIProvider with thinking & logging
core/internal/workflows/manager.go          - Add plan_id to context
core/internal/planner/planner.go            - Add plan_id to context (3 places)
core/internal/router/router.go              - Add plan_id to context
```

## Configuration Reference

### Thinking Mode Types

- **`disabled`** (default) - No thinking, fastest responses
- **`enabled`** with `clear_thinking: true` - Thinking enabled but reset each turn
- **`enabled`** with `clear_thinking: false` - Thinking preserved across turns (for coding/agents)
- **`auto`** - Let the model decide when to use thinking

### Timeout Configuration

Edit `.druppie/config.yaml`:

```yaml
llm:
  timeout_seconds: 120  # Default timeout in seconds
  retries: 3            # Number of retry attempts
```

### API Key Security

**Warning:** The current implementation logs the full API key in each call log. For production:

1. Consider logging only a preview (first 10 characters)
2. Secure the `.druppie` directory with appropriate permissions
3. Implement log rotation and cleanup
4. Consider encrypting sensitive log data

Example to log only API key preview (modify `provider.go`):

```go
"Authorization": fmt.Sprintf("Bearer %s...", p.APIKey[:10])
```
