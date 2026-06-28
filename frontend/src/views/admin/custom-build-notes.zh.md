这个部署当前使用自定义分支，不要用网页里的更新按钮覆盖。

## 当前自定义分支

- 源码目录：`/home/ubuntu/sub2api-src`
- 部署目录：`/home/ubuntu/sub2api-deploy`
- 自定义分支：`custom/subscription-quota-window`
- Fork 仓库：`floating0516/sub2api-ToCreate`
- 官方上游：`Wei-Shaw/sub2api`

## 已修改的问题

### 订阅额度窗口

已修复“订阅有效期按购买时间算，但额度按每天 00:00 重置”的问题。

新创建的订阅现在会从订阅开始时间 `starts_at` 起算额度窗口：

- 日额度：`starts_at` 到 `starts_at + 24h`
- 周额度：`starts_at` 到 `starts_at + 7d`
- 月额度：`starts_at` 到 `starts_at + 30d`

这可以避免下午购买 1 天卡后，因为跨过 00:00 而吃到两次完整日额度。

## 不会自动改变的内容

- 已有旧订阅不会被自动迁移。
- 当前已有的 4 个旧订阅保持原状。
- 只有新分配的订阅会按新的滚动窗口起算。

## 修改过的主要文件

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

## 更新方式

不要点击网页里的更新按钮。网页更新走官方镜像，可能会把自定义修复覆盖掉。

使用部署目录里的脚本更新：

```bash
cd /home/ubuntu/sub2api-deploy
./update-custom-sub2api.sh
```

指定官方版本号：

```bash
./update-custom-sub2api.sh 0.1.140
```

指定应用版本和自定义镜像 tag：

```bash
./update-custom-sub2api.sh 0.1.140 0.1.140-q2
```

脚本会合并官方 `upstream/main`，推送 fork，触发 GitHub Actions 构建自定义镜像，然后只重启 `sub2api` 容器。

## 镜像规则

`docker-compose.yml` 里应该使用固定 tag，不要使用 `latest-custom`。

推荐：

```yaml
image: ghcr.io/floating0516/sub2api-tocreate:0.1.139-q3
```

避免：

```yaml
image: ghcr.io/floating0516/sub2api-tocreate:latest-custom
```

固定 tag 更容易确认当前运行版本，也更方便回滚。

## 回滚方式

更新脚本会备份 compose 文件：

```text
/home/ubuntu/sub2api-deploy/compose-backups/
```

也会记录更新前镜像：

```text
/home/ubuntu/sub2api-deploy/.last-sub2api-image
```

手动回滚时，先查看上一个镜像：

```bash
cat /home/ubuntu/sub2api-deploy/.last-sub2api-image
```

然后把这个镜像写回：

```text
/home/ubuntu/sub2api-deploy/docker-compose.yml
```

最后只重启应用容器：

```bash
cd /home/ubuntu/sub2api-deploy
sudo docker compose pull sub2api
sudo docker compose up -d --no-deps sub2api
curl -fsS http://127.0.0.1:8080/health
```
