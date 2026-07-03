# 月度签到活动方案

版本：v1.0  
日期：2026-07-04  
适用项目：Sub2API 自定义部署

## 1. 方案结论

建议把当前“每日签到给固定余额”的 MVP 升级为“月度连续签到挑战”。核心机制是：

- 每天可签到一次，按 `Asia/Shanghai` 自然日计算。
- 月度周期内展示日历进度、连续天数、累计天数、今日奖励、下一个里程碑。
- 周末奖励显著高于工作日，重点拉动周六/周日活跃。
- 周六、周日发放“当天整天日卡”，有效期固定到次日 00:00，不从用户点击签到的时刻开始滚动 24 小时。
- 连续签到满 7/14/21 天、全勤时给额外奖励。
- 推荐使用“活动额度/到期额度”承接 `$5`、`$10` 这类较大额度，避免永久余额负债过大。

首版推荐规则：

| 场景 | 奖励 | 说明 |
| --- | --- | --- |
| 工作日签到 | `$0.05` 账户余额 | 如果要保守，可沿用当前 `$0.02` |
| 周六签到 | 周六日卡 + `$5` 当日活动额度 | 日卡和额度均到周日 00:00 失效 |
| 周日签到 | 周日日卡 + `$10` 当日活动额度 | 日卡和额度均到周一 00:00 失效 |
| 连续 7 天 | `$1` 账户余额 + 1 张补签卡 | 补签卡用于保护一次断签，不补发当日奖励 |
| 连续 14 天 | `$2` 账户余额 | 可加活动徽章 |
| 连续 21 天 | `$3` 账户余额 | 可加 1 次周末额度加成券 |
| 月度全勤 | `$10` 账户余额 + 7 天体验卡 | 全勤大奖，活动页强提醒 |

## 2. 调研摘要

参考来源：

- 人人都是产品经理《签到体系设计》：常见签到可分每日签到、连续签到、累计签到、混合签到；签到页面要明确入口、状态、成功反馈和未来收益预期。  
  https://www.woshipm.com/pd/4421789.html
- SHEIN 积分签到规则：连续签到奖励随天数递增，7 天为一个小周期，断签会重置。  
  https://ca.shein.com/bonus-point-program-a-371.html
- Microsoft Rewards Streak protection：使用“保护天数”降低断签挫败感，每年有限额。  
  https://support.microsoft.com/en-us/accounts-billing/rewards/microsoft-rewards-streak-protection
- Duolingo Streak 机制：连续天数、里程碑动画、Streak Freeze 能提升习惯养成和留存；其博客提到灵活保护机制有助于用户坚持。  
  https://blog.duolingo.com/how-duolingo-streak-builds-habit/
- Antavo / Open Loyalty 等会员产品资料：游戏化会员常用进度、挑战、徽章、排行榜、奖励目录等机制，并强调目标清晰、奖励清楚、进度可见。  
  https://antavo.com/blog/gamification-in-loyalty-programs/  
  https://www.openloyalty.io/insider/what-is-gamification
- White Label Loyalty 最佳实践：需要监控积分/奖励发放与消耗，并设置清晰过期策略，避免长期成本负债。  
  https://kbase.whitelabel-loyalty.com/product/launch-a-loyalty-program/best-practices-and-tips

对本项目的启发：

- 只给“今日奖励”不够，必须让用户看到“本月还差几天拿大奖”。
- 纯连续签到容易因为一次断签导致放弃，建议同时保留“累计签到保底奖励”。
- `$5`、`$10` 这种大额奖励最好有有效期，优先设计为“活动额度”或“日卡权益”，而不是永久账户余额。
- 周末是活动重点，应在月历上用更明显的视觉标识和奖励预告。

## 3. 活动目标

产品目标：

- 提高用户每日访问频次。
- 让用户在周末也打开产品，并尝试更多 API 能力。
- 给低活跃用户一个低门槛回流入口。
- 让已有付费用户感知平台福利，但不明显稀释付费套餐价值。

业务目标：

- 提升 7 日、14 日、30 日留存。
- 提升周末 DAU 和周末 API 请求用户数。
- 控制活动成本，避免永久余额无限累积。

用户目标：

- 每天有明确收益。
- 周末能拿到明显更高价值的日卡和额度。
- 全勤有可期待的大奖。

## 4. 活动周期

推荐两种周期模式：

### 4.1 自然月模式

- 周期：每月 1 日 00:00:00 至下月 1 日 00:00:00，时区 `Asia/Shanghai`。
- 优点：用户认知简单，月历展示自然。
- 缺点：每月天数不同，全勤门槛 28/29/30/31 天不一致。

### 4.2 30 天挑战模式

- 周期：运营配置任意开始时间，连续 30 个自然日。
- 优点：全勤门槛固定，适合活动运营。
- 缺点：不是自然月，需要页面明确活动日期。

推荐：首期使用“30 天挑战模式”，例如 `2026-08-01 00:00:00` 至 `2026-08-31 00:00:00`。后续稳定后切到自然月常驻。

## 5. 奖励体系

### 5.1 奖励类型定义

| 奖励类型 | 是否推荐 | 用途 | 备注 |
| --- | --- | --- | --- |
| 永久账户余额 | 适合小额 | 工作日基础奖励、里程碑小奖 | 成本会长期沉淀 |
| 活动额度 | 强烈推荐 | 周末 `$5` / `$10` | 需要 `expires_at`，到期自动失效 |
| 日卡订阅 | 强烈推荐 | 周六/周日整天权益 | 通过专用订阅分组发放 |
| 补签卡/保护卡 | 推荐 | 降低断签挫败 | 只保护连续天数，不补发奖励 |
| 徽章/成就 | 推荐 | 低成本精神奖励 | 可用于分享、个人中心展示 |

### 5.2 工作日奖励

默认：

- 周一到周五：`$0.05` 账户余额。
- 如果预算保守：继续用当前 `$0.02`。

原因：

- 工作日奖励要稳定，但不能过大。
- 主要价值来自连续进度和周末预告，而不是单日小额。

### 5.3 周六奖励

周六签到奖励：

- 周六日卡 1 张。
- `$5` 周六活动额度。
- 基础签到计入连续天数和累计天数。

日卡有效期：

- `starts_at`: 周六 00:00:00 `Asia/Shanghai`
- `expires_at`: 周日 00:00:00 `Asia/Shanghai`
- 用户如果在周六 20:00 才签到，仍然到周日 00:00 失效，不顺延到周日 20:00。

注意：当前订阅系统有 `starts_at` 和 `expires_at` 字段，但现有通用分配服务偏向“从当前时间按天数续期”。周末日卡需要新增“按本地自然日精确发放”的专用逻辑。

### 5.4 周日奖励

周日签到奖励：

- 周日日卡 1 张。
- `$10` 周日活动额度。
- 基础签到计入连续天数和累计天数。

日卡有效期：

- `starts_at`: 周日 00:00:00 `Asia/Shanghai`
- `expires_at`: 周一 00:00:00 `Asia/Shanghai`
- 不从点击签到时刻开始计算 24 小时。

### 5.5 连续签到里程碑

| 连续天数 | 奖励 | 设计目的 |
| --- | --- | --- |
| 3 天 | 徽章/动效提示 | 低成本正反馈 |
| 7 天 | `$1` 余额 + 1 张保护卡 | 建立第一周习惯 |
| 14 天 | `$2` 余额 | 中段激励 |
| 21 天 | `$3` 余额 + 周末加成券 | 防止后半程流失 |
| 30 天/全勤 | `$10` 余额 + 7 天体验卡 | 月度大奖 |

保护卡规则：

- 用户漏签 1 天时，可消耗保护卡维持连续天数。
- 保护卡不补发漏签当天的奖励。
- 每个活动周期最多使用 1 张，避免规则被滥用。

### 5.6 累计签到保底

为避免用户漏签后直接放弃，建议同时设置累计奖励：

| 累计签到天数 | 奖励 |
| --- | --- |
| 10 天 | `$1` 账户余额 |
| 20 天 | `$2` 账户余额 |
| 25 天 | `$3` 账户余额 |

连续奖励和累计奖励可以同时获得，但同一里程碑只发一次。

## 6. 成本控制建议

### 6.1 三档预算

保守版：

- 工作日 `$0.02`
- 周六日卡 + `$2` 活动额度
- 周日日卡 + `$3` 活动额度
- 全勤 `$5`

标准版：

- 工作日 `$0.05`
- 周六日卡 + `$5` 活动额度
- 周日日卡 + `$10` 活动额度
- 全勤 `$10`

增长版：

- 工作日 `$0.10`
- 周六日卡 + `$10` 活动额度
- 周日日卡 + `$15` 活动额度
- 全勤 `$20` + 7 天体验卡

推荐先上标准版，但将 `$5` / `$10` 实现为到期活动额度，而不是永久余额。

### 6.2 活动额度过期

建议：

- 周六 `$5`：周日 00:00 失效。
- 周日 `$10`：周一 00:00 失效。
- 里程碑账户余额可以永久有效，但金额较小。

这样用户会在周末即时使用，平台也能控制长期负债。

## 7. 页面方案

### 7.1 用户端页面结构

页面入口：

- 侧边栏保留“每日签到”。
- 首页 Dashboard 增加“今日未签到”小提示。
- 周五晚上到周日可在签到入口显示“周末日卡”标记。

页面模块：

- 顶部：活动名称、今日状态、当前连续天数、累计签到天数。
- 今日奖励卡：明确列出今日可领内容。
- 月历：显示每日已签/未签/周末/里程碑。
- 周末权益卡：展示本周六、周日奖励和失效时间。
- 里程碑进度：7/14/21/全勤。
- 最近记录：展示每次发放的奖励明细。
- 规则说明：时区、日卡有效期、断签、保护卡。

成功反馈：

- 签到成功后弹出奖励明细：
  - 今日已到账。
  - 余额变化。
  - 日卡有效期。
  - 活动额度失效时间。
  - 距离下个里程碑还差几天。

### 7.2 管理端配置

建议新增“签到活动”配置页：

- 活动开关。
- 活动名称。
- 活动开始/结束时间。
- 时区。
- 工作日基础奖励。
- 周六/周日日卡分组 ID。
- 周六/周日活动额度。
- 连续奖励配置。
- 累计奖励配置。
- 保护卡规则。
- 人群限制。

## 8. 后端设计

### 8.1 当前系统基础

当前已经有：

- `user_checkins`：每日签到记录。
- `CheckinService`：按北京时间计算今天、连续天数、写入签到。
- `users.balance`：可加减账户余额。
- `user_subscriptions`：用户订阅，有 `starts_at`、`expires_at`。
- `subscription_plans` / `groups`：可表达日卡/订阅权益。

当前不足：

- 签到奖励是硬编码 `$0.02`。
- 没有活动配置。
- 没有多奖励发放明细。
- 没有活动额度过期模型。
- 现有订阅分配逻辑偏向滚动天数，不适合“周六 00:00 到周日 00:00”的固定自然日卡。

### 8.2 数据表建议

新增活动表：

```sql
CREATE TABLE checkin_campaigns (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    base_weekday_reward DECIMAL(20, 10) NOT NULL DEFAULT 0,
    saturday_group_id BIGINT,
    sunday_group_id BIGINT,
    saturday_bonus_credit DECIMAL(20, 10) NOT NULL DEFAULT 0,
    sunday_bonus_credit DECIMAL(20, 10) NOT NULL DEFAULT 0,
    rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

新增奖励发放明细表：

```sql
CREATE TABLE checkin_reward_grants (
    id BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT NOT NULL REFERENCES checkin_campaigns(id),
    checkin_id BIGINT NOT NULL REFERENCES user_checkins(id),
    user_id BIGINT NOT NULL REFERENCES users(id),
    reward_type VARCHAR(30) NOT NULL,
    reward_key VARCHAR(80) NOT NULL,
    amount DECIMAL(20, 10),
    subscription_id BIGINT,
    starts_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'granted',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (campaign_id, user_id, reward_key)
);
```

可选：如果要实现“活动额度先消耗、到期失效”，新增活动额度账本：

```sql
CREATE TABLE user_expiring_credits (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    source_type VARCHAR(30) NOT NULL,
    source_id BIGINT NOT NULL,
    amount DECIMAL(20, 10) NOT NULL,
    remaining_amount DECIMAL(20, 10) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

如果短期不想改计费链路，可先把周末 `$5` / `$10` 放进专用日卡分组的限额，而不是账户余额。

### 8.3 周末日卡发放逻辑

输入：

- `user_id`
- `campaign_id`
- `local_date`
- `reward_group_id`

计算：

```text
local_start = local_date 00:00:00 Asia/Shanghai
local_end   = local_start + 24h
starts_at   = local_start converted to UTC
expires_at  = local_end converted to UTC
```

规则：

- 如果用户在周六 00:01 签到，过期时间是周日 00:00。
- 如果用户在周六 23:50 签到，过期时间仍是周日 00:00。
- 如果 `now >= expires_at`，不允许补发该日卡，只记录已过期或给兜底小额余额。
- 日卡使用专用分组，避免覆盖用户已有付费订阅。
- 如果用户已有同一专用分组活动订阅：
  - 只允许把 `expires_at` 更新为 `max(existing.expires_at, campaign_day_end)`。
  - 不允许缩短已有订阅。
  - 备注追加活动来源。

### 8.4 API 建议

用户端：

- `GET /api/v1/user/check-in/campaign/status`
  - 返回活动信息、今日奖励、日历、连续天数、累计天数、里程碑。
- `POST /api/v1/user/check-in`
  - 保留现有路径，但返回多奖励明细。
- `GET /api/v1/user/check-in/history`
  - 返回签到记录和奖励明细。

管理端：

- `GET /api/v1/admin/check-in/campaigns`
- `POST /api/v1/admin/check-in/campaigns`
- `PUT /api/v1/admin/check-in/campaigns/:id`
- `POST /api/v1/admin/check-in/campaigns/:id/publish`
- `GET /api/v1/admin/check-in/campaigns/:id/stats`

### 8.5 幂等和并发

必须保证：

- 同一用户同一天只能签到一次。
- 同一奖励只能发放一次。
- 用户重复点击、网络重试不会重复发日卡或余额。
- 签到记录和奖励发放在同一事务内完成。

关键约束：

- `user_checkins (user_id, checkin_date)` 唯一。
- `checkin_reward_grants (campaign_id, user_id, reward_key)` 唯一。

## 9. 风控规则

建议首版加入：

- 只有正常状态用户可参与。
- 新注册用户可参与，但周末高额奖励可设置注册满 24 小时后可领。
- 同一用户每天一次，按账号维度，不按 IP。
- 对异常批量注册、同 IP 多账号、短时间批量领取做后台报表。
- 活动额度不可转赠、不可提现、不可兑换现金。
- 管理员可按用户撤销活动奖励。

## 10. 指标看板

用户指标：

- 每日签到人数。
- 签到转化率：访问签到页后点击签到的比例。
- 连续 3/7/14/21/全勤人数。
- 周六/周日签到人数。
- 断签后回流人数。

成本指标：

- 发放账户余额总额。
- 发放活动额度总额。
- 活动额度实际消耗额。
- 日卡发放数。
- 日卡产生的 API 成本。

业务指标：

- 活动参与用户 7 日留存。
- 活动参与用户周末 API 请求数。
- 活动后付费转化。
- 日卡体验后购买订阅的转化率。

## 11. 上线步骤

### 阶段 1：规则配置化

- 把当前固定 `$0.02` 改为配置项。
- 增加签到奖励明细返回。
- 页面展示今日奖励和到账明细。

### 阶段 2：月度活动

- 增加 `checkin_campaigns`。
- 增加月历视图。
- 支持连续/累计里程碑。
- 支持活动开关。

### 阶段 3：周末日卡

- 新增专用“周六签到日卡”和“周日签到日卡”订阅分组。
- 新增固定自然日发放逻辑。
- 增加奖励发放幂等表。

### 阶段 4：活动额度

- 如果采用永久余额：直接复用 `users.balance`，但周末金额建议保守。
- 如果采用到期额度：增加 `user_expiring_credits`，并修改计费扣减优先级。

### 阶段 5：风控和运营看板

- 管理端活动统计。
- 奖励发放记录查询。
- 异常领取报表。

## 12. 推荐首期配置

活动名：

```text
30 天连续签到挑战
```

活动周期：

```text
2026-08-01 00:00:00 Asia/Shanghai
到
2026-08-31 00:00:00 Asia/Shanghai
```

奖励：

```text
工作日：$0.05 账户余额
周六：周六日卡 + $5 当日活动额度
周日：周日日卡 + $10 当日活动额度
连续 7 天：$1 账户余额 + 1 张保护卡
连续 14 天：$2 账户余额
连续 21 天：$3 账户余额
全勤：$10 账户余额 + 7 天体验卡
累计 10 天：$1 账户余额
累计 20 天：$2 账户余额
累计 25 天：$3 账户余额
```

日卡有效期：

```text
周六日卡：周六 00:00 到周日 00:00
周日日卡：周日 00:00 到周一 00:00
```

## 13. 待确认问题

- `$5` / `$10` 是永久账户余额，还是当日活动额度？
- 周末日卡要使用现有某个订阅分组，还是创建专用活动分组？
- 日卡是否不限量，还是有日限额/模型范围限制？
- 全勤大奖要给 7 天体验卡，还是只给余额？
- 是否需要补签卡？如果需要，每月允许几次？
- 首期活动从 2026-08-01 开始，还是尽快从当前日期启动一个 30 天周期？

