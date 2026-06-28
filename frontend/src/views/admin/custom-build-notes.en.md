This deployment uses a custom branch. Do not overwrite it with the web UI update button.

## Current Custom Branch

- Source directory: `/home/ubuntu/sub2api-src`
- Deployment directory: `/home/ubuntu/sub2api-deploy`
- Custom branch: `custom/subscription-quota-window`
- Fork repository: `floating0516/sub2api-ToCreate`
- Upstream repository: `Wei-Shaw/sub2api`

## What Changed

### Subscription Quota Windows

This build fixes the mismatch where subscription expiry was calculated from the purchase time, but quota reset at midnight.

New subscriptions now anchor quota windows to `starts_at`:

- Daily quota: `starts_at` to `starts_at + 24h`
- Weekly quota: `starts_at` to `starts_at + 7d`
- Monthly quota: `starts_at` to `starts_at + 30d`

This prevents a short subscription from receiving extra full quota windows only because it crosses midnight.

## What Is Not Changed Automatically

- Existing old subscriptions are not migrated automatically.
- The current 4 existing old subscriptions are intentionally left as-is.
- Only newly assigned subscriptions use the rolling quota window.

## Main Files Changed

- `backend/internal/service/subscription_service.go`
- `backend/internal/service/billing_cache_service.go`
- `backend/internal/service/user_subscription_daily_quota_test.go`
- `.github/workflows/custom-image.yml`
- `frontend/src/views/admin/CustomBuildNotesView.vue`
- `frontend/src/views/admin/custom-build-notes.zh.md`
- `frontend/src/views/admin/custom-build-notes.en.md`
- `frontend/src/router/index.ts`
- `frontend/src/components/layout/AppSidebar.vue`
- `frontend/src/i18n/locales/zh.ts`
- `frontend/src/i18n/locales/en.ts`

## Update Method

Do not click the web UI update button. The web update path follows the official image and may replace this custom build.

Use the script in the deployment directory:

```bash
cd /home/ubuntu/sub2api-deploy
./update-custom-sub2api.sh
```

Specify an upstream app version:

```bash
./update-custom-sub2api.sh 0.1.140
```

Specify both app version and custom image tag:

```bash
./update-custom-sub2api.sh 0.1.140 0.1.140-q2
```

The script merges `upstream/main`, pushes the fork branch, triggers GitHub Actions, builds the custom image, and restarts only the `sub2api` container.

## Image Rule

Use a fixed tag in `docker-compose.yml`, not `latest-custom`.

Good:

```yaml
image: ghcr.io/floating0516/sub2api-tocreate:0.1.139-q3
```

Avoid:

```yaml
image: ghcr.io/floating0516/sub2api-tocreate:latest-custom
```

Fixed tags are easier to audit and roll back.

## Rollback

The update script backs up the compose file:

```text
/home/ubuntu/sub2api-deploy/compose-backups/
```

It also records the previous image:

```text
/home/ubuntu/sub2api-deploy/.last-sub2api-image
```

For a manual rollback, inspect the previous image:

```bash
cat /home/ubuntu/sub2api-deploy/.last-sub2api-image
```

Then set it in:

```text
/home/ubuntu/sub2api-deploy/docker-compose.yml
```

Restart only the app container:

```bash
cd /home/ubuntu/sub2api-deploy
sudo docker compose pull sub2api
sudo docker compose up -d --no-deps sub2api
curl -fsS http://127.0.0.1:8080/health
```
