# Week 2, Step 5: GitHub Actions CI/CD Setup

**Status**: Implementation Phase  
**Date**: 2025-11-10  
**Objective**: Verify and test GitHub Actions CI/CD pipeline  
**Timeline**: 1-2 hours

## Overview

GitHub Actions CI/CD pipeline is already configured. This step verifies it's working correctly and all tests pass.

## Current Status

### ✅ Workflow File Created
- `.github/workflows/integration-tests.yml` - Complete workflow definition
- Triggers: PR, nightly schedule (2 AM UTC), manual dispatch
- Jobs: integration-tests, unit-tests

### ✅ Integration Tests Exist
- `internal/services/ad_conversion_integration_test.go` - 468 lines
- Tests ayeT sandbox connectivity
- Tests offerwall callbacks
- Tests S2S conversion flow

### ✅ Unit Tests Exist
- `internal/services/ad_conversion_test.go` - Unit test coverage
- Codecov integration configured

## Workflow Configuration

### Triggers
```yaml
on:
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 2 * * *'  # Nightly at 2 AM UTC
  workflow_dispatch:     # Manual trigger
```

### Jobs

#### Job 1: Integration Tests
- PostgreSQL 15 service container
- Go 1.21 environment
- Database migrations
- Bot startup
- Integration test execution
- Discord notifications on failure

#### Job 2: Unit Tests
- Go 1.21 environment
- Unit test execution
- Coverage reporting to Codecov

## Required GitHub Secrets

The workflow requires these secrets to be configured in GitHub:

```
AYET_API_KEY_SANDBOX          # ayeT sandbox API key
AYET_CALLBACK_TOKEN_SANDBOX   # ayeT callback token
DISCORD_TOKEN_TEST            # Test Discord bot token
SENTRY_DSN_TEST               # Test Sentry DSN
DISCORD_WEBHOOK_CI            # Discord webhook for CI notifications
```

## Step 5 Tasks

### Task 1: Verify Workflow File
- [x] Workflow file exists
- [x] Syntax is valid
- [x] All required fields present
- [x] Triggers configured correctly

### Task 2: Configure GitHub Secrets
- [ ] Add AYET_API_KEY_SANDBOX
- [ ] Add AYET_CALLBACK_TOKEN_SANDBOX
- [ ] Add DISCORD_TOKEN_TEST
- [ ] Add SENTRY_DSN_TEST
- [ ] Add DISCORD_WEBHOOK_CI

### Task 3: Verify Integration Tests
- [ ] Tests compile without errors
- [ ] Tests can connect to PostgreSQL
- [ ] Tests can start bot
- [ ] Tests can call endpoints
- [ ] Tests verify results

### Task 4: Run Workflow Manually
- [ ] Trigger workflow via GitHub UI
- [ ] Monitor workflow execution
- [ ] Verify all steps pass
- [ ] Check test results
- [ ] Verify Discord notification

### Task 5: Verify Test Coverage
- [ ] Unit tests pass
- [ ] Coverage > 80%
- [ ] Codecov integration working
- [ ] Coverage reports generated

## How to Configure GitHub Secrets

1. Go to GitHub repository settings
2. Navigate to **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Add each secret:

```
Name: AYET_API_KEY_SANDBOX
Value: [your-sandbox-api-key]

Name: AYET_CALLBACK_TOKEN_SANDBOX
Value: [your-sandbox-callback-token]

Name: DISCORD_TOKEN_TEST
Value: [your-test-bot-token]

Name: SENTRY_DSN_TEST
Value: https://[key]@sentry.io/[project-id]

Name: DISCORD_WEBHOOK_CI
Value: https://discord.com/api/webhooks/[webhook-id]/[webhook-token]
```

## How to Run Workflow Manually

1. Go to GitHub repository
2. Click **Actions** tab
3. Select **Integration Tests** workflow
4. Click **Run workflow** button
5. Select branch (main)
6. Click **Run workflow**

## Workflow Execution Steps

### Integration Tests Job

```
1. Checkout code
2. Set up Go 1.21
3. Download dependencies
4. Run database migrations
5. Start AGIS Bot in background
6. Run integration tests
7. Upload test results
8. Stop AGIS Bot
9. Check metrics endpoint
10. Notify Discord on failure
```

### Unit Tests Job

```
1. Checkout code
2. Set up Go 1.21
3. Download dependencies
4. Run unit tests with coverage
5. Upload coverage to Codecov
```

## Expected Test Results

### Integration Tests
- ✅ TestAyetSandboxConnection - Verify sandbox connectivity
- ✅ TestAyetOfferwallCallback - Test offerwall conversion
- ✅ TestAyetSurveywallCallback - Test surveywall conversion
- ✅ TestAyetVideoCallback - Test video placement
- ✅ TestS2SCallbackValidation - Verify callback signature
- ✅ TestConversionTracking - Verify conversion tracking
- ✅ TestErrorHandling - Test error scenarios
- ✅ TestMetricsCollection - Verify metrics

### Unit Tests
- ✅ All unit tests pass
- ✅ Coverage > 80%
- ✅ No race conditions
- ✅ All edge cases covered

## Troubleshooting

### Workflow Not Triggering
- Check branch protection rules
- Verify workflow file syntax
- Check if workflow is enabled
- Review GitHub Actions logs

### Tests Failing
- Check PostgreSQL service health
- Verify bot startup logs
- Check database migrations
- Review test output

### Secrets Not Found
- Verify secret names match exactly
- Check secret values are correct
- Verify repository has access to secrets
- Check organization-level secrets

### Discord Notifications Not Working
- Verify webhook URL is correct
- Check webhook permissions
- Verify Discord server is accessible
- Check webhook is not expired

## Monitoring Workflow

### GitHub Actions Dashboard
- Go to **Actions** tab
- View workflow runs
- Click on run to see details
- Check individual step logs

### Discord Notifications
- Check #alerts-ci channel
- Verify message format
- Check for any errors
- Review failure details

### Codecov Reports
- Go to Codecov.io
- View coverage trends
- Check coverage by file
- Review coverage changes

## Next Steps After Step 5

Once GitHub Actions is verified:
1. Proceed to Step 6: A/B Testing Verification
2. Create test A/B experiment
3. Verify sticky assignments
4. Test experiment results

## Files Involved

### Workflow
- `.github/workflows/integration-tests.yml` - Main workflow

### Tests
- `internal/services/ad_conversion_integration_test.go` - Integration tests
- `internal/services/ad_conversion_test.go` - Unit tests

### Configuration
- `go.mod` - Go dependencies
- `go.sum` - Dependency checksums
- `deployments/migrations/v2.0-production-enhancements.sql` - Database schema

## Success Criteria

- [x] Workflow file exists and is valid
- [ ] All GitHub secrets configured
- [ ] Integration tests pass
- [ ] Unit tests pass
- [ ] Coverage > 80%
- [ ] Discord notifications working
- [ ] Codecov integration working

## Estimated Time

- Configure secrets: 10 minutes
- Run workflow: 5-10 minutes
- Review results: 5 minutes
- **Total: 20-25 minutes**

## Resources

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GitHub Secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [Codecov Integration](https://codecov.io/github)
- [Integration Tests Guide](docs/INTEGRATION_TESTS.md)

---

**Step 5 Ready to Execute!** 🚀
