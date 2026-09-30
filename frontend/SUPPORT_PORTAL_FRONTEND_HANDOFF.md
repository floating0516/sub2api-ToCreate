# 工单支持中心前端交接

## 2026-09-30 后端契约核对与前端接线

本轮只修改前端及其远端测试清单，不修改 Agent、BFF、数据库、部署脚本、
菜单或客服权限体系。合并及部署仍由负责窗口安排；本轮不部署 18080/8080。

已核对最终实现及其 GitHub Actions CI：

- Agent：`e33f1f9f5b160e0f1b412e8e84f7dea4a3c2de25`，
  `feature/ticket-portal-backend-20260930`，CI `36669913074` 成功。
- BFF：`b9a69257b187cc6f09c01cfaac1b73fe7d60154e`，
  `feature/support-portal-backend-20260930`，CI `36671272921` 成功。
- 契约：`/home/ubuntu/sub2api-support-agent/docs/ticket-portal-api-contract.md`。
  旧 `ticket-portal-api-gaps.md` 已过时，不能作为最新实现清单。

### 本轮代码能力（尚未部署验收）

- 列表、全量统计、搜索防抖、类型/状态筛选和分页使用同源 support API。
- 独立创建表单绑定与长度校验，订单问题必须关联本人代充订单；复用现有
  `listManagedRechargeOrders(100)`，不读取或展示订单凭证/兑换码。
- 标题 4-200 字，描述 10-8000 字，产品名 1-128 字；新单默认普通优先级。
- 详情正文与回复分开，区分用户、客服和系统消息；未知角色不冒充客服。
- 解决、关闭、重新打开须确认，并依据 `allowed_actions` 显示入口。
  已解决/关闭工单即使错误返回 `reply` 权限，也不会启用回复。
- 上传复用 BFF 私有上传接口，检查大小、扩展名、类型及剩余附件数量；
  失败文件可重试或移除，未完成上传不允许提交。下载经带认证的共享
  `apiClient` 获取 Blob，不直接跟随响应提供的 URL。
- 创建/回复同一内容失败重试复用 UUID 幂等键；修改内容后使用新键。
  失败保留输入，只有明确成功响应才提示成功。
- 切换账号清除表单、列表、统计、详情及订单选项，忽略旧账号晚到响应。
- 保留 AI 咨询、引用、草稿确认/取消及单张旧工单兼容路径。

### 运行时和未完成边界

- 列表 404 时继续显示“未接通”，统计为 `--`，不启用直接提交。
  503/超时等错误显示失败和重试，不当成没有工单。
- 详情没有 `replies` / `allowed_actions` 时只展示旧表头，不假装能回复。
- 列表摘要接口没有最新回复正文，所以行摘要使用真实问题描述，
  不冒充“最新消息”；后端补字段后再适配。
- 订单选择只涵盖现有接口返回的最近 100 条 managed-recharge 订单。
  不支持另一套支付订单体系。
- 本轮未实现管理员客服工作台及优先级调整入口；未新增相关导航。
- 未执行 Agent PostgreSQL 集成测试，未进行真实接口浏览器流程、
  跨用户越权及新代码移动端截图验收；这些必须在后端和前端整合到
  18080 后完成，不能用模拟接口单测代替。

### 检查与交付

- 前端分支：`feature/support-portal-frontend-20260930`。
- 用户端接线代码：`31995b752ac16a45023a72f71af451634265ae8c`。
- 最终代码及 AI 加载隔离修正：`9d85692af0ff9acfc2d88f98ceb90b5e542da565`。
- 最终代码 CI：[36686447758](https://github.com/floating0516/sub2api-ToCreate/actions/runs/36686447758)。
  `frontend` job 的 lint、typecheck 成功，41 个测试文件、452 项测试通过。
- 最终代码安全扫描：[36686447864](https://github.com/floating0516/sub2api-ToCreate/actions/runs/36686447864)，成功。
- `9d85692` 之后只补充此交接文档，不改变上述已验证代码。

本轮新增/改造的文件组：

- `frontend/src/views/user/SupportView.vue`：页面编排及已有订单选择接入。
- `frontend/src/components/support/SupportTicket{CreateDialog,Discussion,List,Row}.vue`：表单、消息、列表和工单行。
- `frontend/src/components/support/SupportAttachment{Picker,s}.vue`：私有上传与授权下载。
- `frontend/src/components/support/ticketPresentation.ts`：类型、状态、角色与日期展示。
- `frontend/src/composables/useTicketPortal.ts`：真实查询、权限操作、失效响应和账号隔离。
- `frontend/src/api/support.ts`、中文/英文 support locale、相应测试及 `Makefile`。

本机只做 `git diff --check` 和源码/配置阅读，不安装依赖、运行构建或测试。
新增 API、composable、表单/详情/列表、附件测试已加入
`FRONTEND_CRITICAL_VITEST`，随 GitHub Actions 的 `make test-frontend`
运行 lint、typecheck 及关键测试。本轮远端结果以新前端提交的 CI 为准，
不借用下述旧布局或后端提交的 CI 结果。

## 已完成的布局预览（历史）

`9b59372`、`98ebdcc` 已通过远端前端检查（37 文件、403 测试），并以
`0.2.10-tc1.51-rc.4` 发布到 18080，桌面/手机布局及创建弹窗截图已检查。
生产仍未改动。部署详情以 `sub2api-deploy/custom/CUSTOM_BUILD_DETAILS.md`
为准。下文保留最初布局交接范围，其中“尚未验证”描述只针对当时快照。

## 最初布局范围（历史）

基于 `feature/support-console-v1-0.2.10`，基线提交为
`b0de60e02b3fb4912045c3c81ad3696af5995d0c`。工单后端缺口仍待另一个窗口接入。

用户已另行授权远程构建本轮前端快照，并发布到 `18080` 查看布局。
使用独立预览分支 `preview/support-portal-layout-20260930`，不合并主线，
不移动 `latest-custom`，不推广正式服务。实际构建和部署结果见部署目录记录。

最初布局的接口边界以支持服务的 `docs/ticket-portal-api-gaps.md` 为准，本机文件位于
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

## 最初接入条件（已由后端窗口交付）

由后端窗口确认并实现列表/统计、详情时间线、回复以及直接创建的最终契约，
控制台 BFF 同步转发。浏览器不直接连接支持 Agent，不传用户 ID 指定归属。
明确类别字段、状态枚举、可执行操作、分页与错误结构；创建和回复需要幂等设计。
订单、上传和客服权限应沿用现有体系，本轮没有新增替代体系。

现有 API 对自己的单张工单/会话进行权限检查；本轮没有实现客服工作台，
也没有修改 Go BFF、支持服务或数据库。

## 最初验证计划（实际结果见上文）

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
