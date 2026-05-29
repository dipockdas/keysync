# Integration Test Guide - First-Party Platform Configs

This guide walks through end-to-end testing of the first-party platform configurations (`docs/platform-configs/*.json`) against real platform APIs.

## Prerequisites

You'll need active accounts and API tokens for each platform you want to test:

### Vercel
1. Sign up at https://vercel.com
2. Create a test project (or use an existing one)
3. Get project ID: Run `vercel project ls` or check URL (e.g., `prj_abc123xyz`)
4. Generate API token: https://vercel.com/account/tokens → "Create Token"
5. Store token: `keysync set VERCEL_TOKEN=<your-token>`

### Railway
1. Sign up at https://railway.app
2. Create a test project
3. Get service ID: Click project → Settings → Project ID
4. Get environment: Usually "production" (check project settings)
5. Generate API token: https://railway.app/account/tokens → "Create Token"
6. Store token: `keysync set RAILWAY_TOKEN=<your-token>`

### Supabase
1. Sign up at https://supabase.com
2. Create a test project
3. Get project ref: Project Settings → General → Reference ID
4. Generate API token: https://supabase.com/dashboard/account/tokens → "Generate new token"
5. Store token: `keysync set SUPABASE_TOKEN=<your-token>`

## Setup Test Project

Create a test `.keysync.json` with all three platforms:

```json
{
  "repos": {
    "test/integration": {
      "project": "keysync-integration-test",
      "globals": ["TEST_SECRET"],
      "platforms": {
        "vercel": {
          "type": "http",
          "endpoint": "https://api.vercel.com/v9/projects/{PROJECT_ID}/env",
          "method": "POST",
          "token_env": "VERCEL_TOKEN",
          "headers": {
            "Authorization": "Bearer {TOKEN}",
            "Content-Type": "application/json"
          },
          "body": {
            "key": "{KEY}",
            "value": "{VALUE}",
            "target": ["production", "preview"],
            "type": "encrypted"
          },
          "template_vars": {
            "PROJECT_ID": "YOUR_VERCEL_PROJECT_ID"
          }
        },
        "railway": {
          "type": "http",
          "endpoint": "https://backboard.railway.app/graphql/v2",
          "method": "POST",
          "token_env": "RAILWAY_TOKEN",
          "headers": {
            "Authorization": "Bearer {TOKEN}",
            "Content-Type": "application/json"
          },
          "body": {
            "query": "mutation($input: VariableUpsertInput!) { variableUpsert(input: $input) { id name } }",
            "variables": {
              "input": {
                "name": "{KEY}",
                "value": "{VALUE}",
                "projectId": "{SERVICE_ID}",
                "environment": "{ENVIRONMENT}"
              }
            }
          },
          "template_vars": {
            "SERVICE_ID": "YOUR_RAILWAY_SERVICE_ID",
            "ENVIRONMENT": "production"
          }
        },
        "supabase": {
          "type": "http",
          "endpoint": "https://api.supabase.com/v1/projects/{REF}/secrets",
          "method": "POST",
          "token_env": "SUPABASE_TOKEN",
          "headers": {
            "Authorization": "Bearer {TOKEN}",
            "Content-Type": "application/json"
          },
          "body": [
            {
              "name": "{KEY}",
              "value": "{VALUE}"
            }
          ],
          "template_vars": {
            "REF": "YOUR_SUPABASE_PROJECT_REF"
          }
        }
      }
    }
  }
}
```

## Test Procedure

### 1. Store test secret locally

```bash
keysync set -p keysync-integration-test TEST_SECRET=test_value_$(date +%s)
```

### 2. Verify secret is stored

```bash
keysync get -p keysync-integration-test TEST_SECRET
```

Expected: Should print the value you just set.

### 3. Push to all platforms

```bash
keysync push --project keysync-integration-test
```

Expected output:
```
Pushing to GitHub Secrets...
✓ Pushed TEST_SECRET to GitHub

Pushing to vercel...
✓ Pushed TEST_SECRET to vercel

Pushing to railway...
✓ Pushed TEST_SECRET to railway

Pushing to supabase...
✓ Pushed TEST_SECRET to supabase

✓ Successfully pushed 1 secret to 4 platforms
```

### 4. Verify on platforms

**Vercel**:
```bash
# CLI verification
vercel env ls

# Or check web UI:
# https://vercel.com/your-org/your-project/settings/environment-variables
```

**Railway**:
```bash
# Check web UI:
# https://railway.app/project/your-project → Variables tab
```

**Supabase**:
```bash
# CLI verification (if installed)
supabase secrets list --project-ref YOUR_REF

# Or check web UI:
# https://supabase.com/dashboard/project/your-ref/settings/vault
```

### 5. Test individual platform push

```bash
# Push to only Vercel
keysync push --project keysync-integration-test --platforms vercel

# Push to only Railway
keysync push --project keysync-integration-test --platforms railway

# Push to only Supabase
keysync push --project keysync-integration-test --platforms supabase
```

### 6. Test with special characters

Test that percent encoding works correctly for keys with special characters:

```bash
keysync set -p keysync-integration-test "MY_KEY_WITH_UNDERSCORES=test_value"
keysync set -p keysync-integration-test "KEY-WITH-DASHES=test_value"
keysync push --project keysync-integration-test
```

Verify these appear correctly in each platform's dashboard.

## Troubleshooting

### Authentication errors

If you see `401 Unauthorized`:
- Check token is stored: `keysync get VERCEL_TOKEN` (or RAILWAY_TOKEN, SUPABASE_TOKEN)
- Verify token is valid on platform (regenerate if needed)
- Check token has correct permissions

### Template variable errors

If you see `404 Not Found` or `Invalid project ID`:
- Double-check `PROJECT_ID`, `SERVICE_ID`, `REF` in `.keysync.json`
- Verify IDs match your actual platform resources

### Timeout errors

If you see `context deadline exceeded`:
- Check network connectivity
- Verify platform API is accessible
- Default timeout is 30 seconds (should be sufficient for all platforms)

### Body structure errors

If you see `400 Bad Request`:
- The generic engine might have serialized the body incorrectly
- Check platform API docs to ensure body matches expected structure
- Enable debug logging: `keysync push --project keysync-integration-test --verbose`

## Cleanup

After testing, remove the test secret from all platforms:

```bash
# Delete locally
keysync rm -p keysync-integration-test TEST_SECRET

# Delete from platforms (manual cleanup recommended)
# Vercel: Web UI → Environment Variables → Delete
# Railway: Web UI → Variables → Delete
# Supabase: Web UI → Vault → Delete
```

## Expected Results

All tests should pass with:
- ✅ Secrets successfully stored in OS keychain
- ✅ Secrets successfully pushed to all platforms
- ✅ Secrets visible in platform dashboards with correct values
- ✅ Special characters handled correctly (underscores, dashes)
- ✅ No authentication errors
- ✅ No timeout errors
- ✅ Body structures accepted by platform APIs

## Next Steps

Once integration tests pass:
1. Update CHANGELOG.md with platform config validation
2. Add integration test results to CI/CD documentation
3. Consider adding automated integration tests (if platform test accounts available)
