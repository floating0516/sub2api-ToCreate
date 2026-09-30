# 工单支持中心前端交接

## 本轮范围

基于 `feature/support-console-v1-0.2.10`，基线提交为
`b0de60e02b3fb4912045c3c81ad3696af5995d0c`。工单后端缺口仍待另一个窗口接入。

用户已另行授权远程构建本轮前端快照，并发布到 `18080` 查看布局。
使用独立预览分支 `preview/support-portal-layout-20260930`，不合并主线，
不移动 `latest-custom`，不推广正式服务。实际构建和部署结果见部署目录记录。

接口边界以支持服务的 `docs/ticket-portal-api-gaps.md` 为准，本机文件位于
`/home/ubuntu/sub2api-support-agent/docs/ticket-portal-api-gaps.md`。
该文档是缺口清单，其中建议路径并非已实现接口。

- 保留 `/support` 路由、现有导航、功能开关、AppLayout、品牌色及技术栈。
- 新增“我的工单”布局、五种状态展示位置、工单记录与 AI 咨询切换。
- 工单完整列表仍显示“尚未接通”，统计显示 `--`，不伪造零条或完整总数。
- 当前会话关联的单张工单单独展示；`submitted` 只显示“已提交”。
- 复用 BaseDialog、SearchInput、Select、Icon、现有 support API/Store 和提示消息。
- 恢复 AI 问答、证据引用、草稿人工确认/取消；不自动确认草稿。
- 表单直接创建、搜索/筛选/分页、附件、人工时间线及状态操作尚未接通。
- 后续 AI 对话不是工单回复，工单类型不从 `product` 推断。
- 确认操作返回明确成功标记；失败或缺少工单号时保留草稿，不提示成功。
- 切换账号清空会话缓存状态；忽略旧账号晚到的响应。后端权限仍是最终边界。

## 涉及文件

- `frontend/src/views/user/SupportView.vue`
- `frontend/src/components/support/SupportConversation.vue`
- `frontend/src/api/support.ts`
- `frontend/src/stores/support.ts`
- `frontend/src/i18n/locales/{zh,en}/support.ts`
- `frontend/src/stores/__tests__/support.spec.ts`
- `frontend/src/views/user/__tests__/SupportView.spec.ts`
- `frontend/src/components/support/__tests__/SupportConversation.spec.ts`
- `Makefile`：把上述三组测试加入现有前端关键测试清单。
- 本交接文件。

## 后续接入条件

由后端窗口确认并实现列表/统计、详情时间线、回复以及直接创建的最终契约，
控制台 BFF 同步转发。浏览器不直接连接支持 Agent，不传用户 ID 指定归属。
明确类别字段、状态枚举、可执行操作、分页与错误结构；创建和回复需要幂等设计。
订单、上传和客服权限应沿用现有体系，本轮没有新增替代体系。

现有 API 对自己的单张工单/会话进行权限检查；本轮没有实现客服工作台，
也没有修改 Go BFF、支持服务或数据库。

## 验证与合并门槛

本机只运行轻量差异/配置检查。按照 `sub2api-upgrade`，禁止在此 VPS
安装依赖、编译、运行测试套件或开发服务器。本轮新增测试尚未执行，
不要把已有其他提交的 CI 结果作为当前改动的验证结果。

本轮 `git diff --check` 已通过；`make -n test-frontend-critical` 仅打印执行计划，
确认三组支持中心测试已进入远程检查清单，没有在本机执行测试命令。

另一个窗口收取改动并按约定提交/推送后，应使用现有 GitHub Actions CI
的 `frontend` job 运行 `make test-frontend`，包含 lint、typecheck 和关键测试。
镜像编译走现有 Custom Docker Image 工作流，使用精确提交、tc-style 候选标签，
保持 `DEPLOY=0` / 不部署，并设置 `push_latest_custom=false`。

远程编译及测试通过后，另行安排 `18080` 浏览器验收。至少检查桌面与手机
布局、长标题、弹窗、AI 发送失败重试、证据引用、草稿确认/取消与账号切换。
本轮尚未做浏览器截图或移动端视觉验收，不启动本机开发服务器作为替代。
后续主线合并和正式部署仍由负责窗口安排；本次仅允许更新 `18080`，不改 `8080`。
