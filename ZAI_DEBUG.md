# Z.AI Debugging Guide

## Changes Made

I've enhanced the logging in `core/internal/llm/provider.go` to help diagnose the timeout issues:

### 1. Enhanced Z.AI Provider Logging (lines 763-857)
- Logs the URL being called
- Logs request body size
- Logs prompt and system prompt lengths
- Logs the model being used
- Logs API key preview (first 10 chars)
- **Logs request timing** - when request starts and how long it takes
- Logs response status and timing
- Logs error response bodies
- Logs token usage and response content length
- Logs total request time and estimated cost

### 2. Enhanced Retry Logic Logging (lines 228-287)
- Logs which provider is being used (ZAI, Gemini, etc.)
- Logs max retries and timeout settings
- Logs prompt sizes
- Logs each attempt with timing
- Logs success/failure with elapsed time

## What the Logs Will Tell You

When you rebuild and restart the container, you'll now see detailed logs like:

```
[LLM] Starting generation with provider: ZAI (max retries: 3, timeout: 2m0s)
[LLM] Prompt size: 1234 chars, System prompt: 567 chars
[ZAI] Sending request to https://api.z.ai/api/coding/paas/v4/chat/completions
[ZAI] Request body size: 2345 bytes
[ZAI] Prompt length: 1234 chars, System prompt length: 567 chars
[ZAI] Model: GLM-4.7
[ZAI] Using API key (first 10 chars): abcdef1234...
[ZAI] Response received in 1.5s (status: 200)
[ZAI] Response: 1 choices, 1234 prompt tokens, 567 completion tokens
[ZAI] Response content length: 2345 chars
[ZAI] Total request time: 1.5s, Estimated cost: 0.000234 EUR
[LLM] Success on attempt 1 after 1.5s
```

Or if it fails:

```
[ZAI] Request failed after 2m0s: Post "https://api.z.ai/...": context deadline exceeded
[LLM] Attempt 1 failed after 2m0s: zai request failed: context deadline exceeded. Retrying in 2s...
```

## Troubleshooting Steps

### 1. Rebuild and Restart

```bash
# From the repository root
docker-compose down
docker-compose build --no-cache druppie
docker-compose up
```

### 2. Check the Logs

Watch the logs in real-time:

```bash
docker-compose logs -f druppie
```

### 3. Common Issues and Solutions

#### Issue: Context Deadline Exceeded (timeout)

**Possible Causes:**

1. **API is genuinely slow** - The Z.AI API is taking longer than 120 seconds to respond
2. **Network connectivity from Docker** - The container has issues reaching the API
3. **DNS resolution issues** - The container can't resolve the API hostname
4. **Firewall/Proxy issues** - Outbound connections are being blocked or slowed

**Solutions:**

**A. Test connectivity from inside the container:**

```bash
# Open a shell in the running container
docker-compose exec druppie sh

# Test DNS resolution
nslookup api.z.ai

# Test connectivity
curl -v https://api.z.ai/api/coding/paas/v4/chat/completions

# Test with a simple request
time curl -X POST https://api.z.ai/api/coding/paas/v4/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"model":"GLM-4.7","messages":[{"role":"user","content":"test"}],"stream":false}'
```

**B. Increase timeout temporarily:**

Edit `.druppie/config.yaml`:

```yaml
llm:
  timeout_seconds: 300  # Increase from 120 to 300 seconds (5 minutes)
```

**C. Check Docker network settings:**

The container uses bridge networking by default. If you're behind a corporate proxy or firewall, you may need to:

```yaml
# In docker-compose.yml
services:
  druppie:
    # ... existing config ...
    extra_hosts:
      - "api.z.ai:<actual IP address>"  # Bypass DNS if needed
    dns:
      - 8.8.8.8
      - 8.8.4.4
```

**D. Check if API key is valid:**

The logs will show the first 10 characters of your API key. Verify this matches your actual key.

#### Issue: Connection Refused

This usually means the API is unreachable from the container. Check:
- Can you access the API from your host machine?
- Is there a firewall blocking the container?
- Does your company require a proxy for outbound connections?

#### Issue: Very Slow Responses

If the API responds but takes a long time (>30 seconds):
1. Check your network bandwidth
2. Check if the Z.AI service is having issues (check their status page)
3. Try a different provider to see if it's specific to Z.AI

### 4. Quick Tests

**Test with a different provider:**

Edit `.druppie/config.yaml`:

```yaml
llm:
  default_provider: gemini  # or ollama if you have it locally
```

This will help determine if the issue is specific to Z.AI or a general network problem.

**Test from host machine:**

```bash
# If you have the CLI built
./druppie run "say hello"
```

Compare the timing between the host and the container.

### 5. Getting More Diagnostic Information

If you need even more detail, you can:

1. **Enable HTTP tracing** - Add to the provider code:
   ```go
   import "net/http/httptrace"
   ```

2. **Capture network traffic**:
   ```bash
   docker-compose exec druppie tcpdump -i eth0 -w /tmp/capture.pcap host api.z.ai
   ```

3. **Check Docker logs**:
   ```bash
   docker inspect druppie | grep -A 20 "NetworkSettings"
   ```

## Next Steps

1. **Rebuild the container** with the new logging
2. **Trigger a request** and capture the logs
3. **Share the logs** so we can see:
   - How long the request takes before timing out
   - Whether the API is responding at all
   - What the actual error message is

The logs will clearly show if the issue is:
- Network connectivity (immediate failure)
- DNS resolution (immediate failure)
- API slowness (takes >120 seconds)
- API errors (non-200 status codes)
