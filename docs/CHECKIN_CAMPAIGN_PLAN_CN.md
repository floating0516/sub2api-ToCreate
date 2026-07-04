# 月度签到活动方案

版本：v1.3  
日期：2026-07-04  
适用项目：Sub2API 自定义部署

## 1. 方案结论

建议把签到活动定义成两个独立权益池：

- 签到额度：每个用户每月最多约 `$20`，全部是限时活动额度，不发永久额度。
- 周末日卡：周六、周日签到发当天整天日卡，月度感知价值约 `$20`，不进入额度钱包。
- 全部签到额度统一到“下月末”到期，技术上按 `Asia/Shanghai` 的次月 1 日 00:00:00 失效。
- 周末日卡按自然日生效：周六卡到周日 00:00 失效，周日卡到周一 00:00 失效，不从签到时刻滚动 24 小时。
- 页面需要明确拆分“签到额度”和“周末日卡”，避免用户误以为日卡也会折算成余额。
- 不再写入 `users.balance`。当前 `$0.02` 永久余额测试逻辑后续需要替换为到期活动额度。

首期推荐规则：

| 场景 | 奖励 | 说明 |
| --- | --- | --- |
| 每日签到 | `$0.25` 签到额度 | 每天一次，计入 `$20` 月度上限 |
| 首次参与 | `$1` 签到额度 | 只发一次，计入 `$20` 月度上限 |
| 周六签到 | 周六整天日卡 | 有效期到周日 00:00，不额外发周末额度 |
| 周日签到 | 周日整天日卡 | 有效期到周一 00:00，不额外发周末额度 |
| 连续 3 天 | `$0.50` 签到额度 + 徽章 | 早期正反馈 |
| 连续 7 天 | `$1` 签到额度 + 1 张保护卡 | 保护卡只保连续天数，不补发漏签奖励 |
| 连续 14 天 | `$1.50` 签到额度 | 中段激励 |
| 连续 21 天 | `$2` 签到额度 | 防止后半程流失 |
| 累计 10 天 | `$1` 签到额度 | 漏签用户也有目标 |
| 累计 20 天 | `$2` 签到额度 | 累计保底 |
| 累计 25 天 | `$2` 签到额度 | 临近月底保底 |
| 月度全勤 | 补足到 `$20` 签到额度上限 + 全勤徽章 | 不突破 `$20` 签到额度预算 |

按 30 天全勤估算：

```text
每日签到：30 * $0.25 = $7.50
连续奖励：$0.50 + $1 + $1.50 + $2 = $5.00
累计奖励：$1 + $2 + $2 = $5.00
小计：$17.50
全勤补足：最多补到 $20
```

如果用户有首次参与 `$1`，也计入 `$20` 封顶；全勤补足金额会相应减少。

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

- 只展示“今日奖励”不够，页面要让用户看到本月 `$20` 签到额度进度和周末日卡权益。
- 纯连续签到容易因为一次断签导致放弃，所以保留累计 10/20/25 天的保底奖励。
- 大额福利应该用到期额度和日卡权益承接，不进入永久余额。
- 周末日卡是活动亮点，应在日历和签到成功反馈里强提示。

## 3. 活动目标

产品目标：

- 提高用户每日访问频次。
- 让用户在周末也打开产品，并尝试更多 API 能力。
- 给低活跃用户一个低门槛回流入口。
- 让已有付费用户感知平台福利，但不稀释付费套餐价值。

业务目标：

- 提升 7 日、14 日、30 日留存。
- 提升周末 DAU 和周末 API 请求用户数。
- 控制活动成本，避免永久余额长期沉淀。

用户目标：

- 每天有明确收益。
- 签到额度有足够长的使用期：当月领取，下月末到期。
- 周末能拿到明确的整天日卡权益。
- 全勤有可期待的补足奖励。

## 4. 活动周期和到期规则

推荐首期使用自然月模式：

- 活动周期：每月 1 日 00:00:00 到下月 1 日 00:00:00，时区 `Asia/Shanghai`。
- 签到额度到期：领取月份的下一个自然月末到期。
- 技术表达：`expires_at` 为领取月份后第二个月 1 日 00:00:00 `Asia/Shanghai`。

示例：

```text
2026-08-01 至 2026-08-31 领取的签到额度
统一在 2026-10-01 00:00:00 Asia/Shanghai 失效
用户侧文案：有效期至 2026-09-30 23:59
```

这样用户有 1-2 个月的使用窗口，既不太短，也不会形成永久负债。

## 5. 奖励体系

### 5.1 奖励资产分层

| 奖励类型 | 是否进入钱包额度 | 用途 | 到期规则 |
| --- | --- | --- | --- |
| 签到额度 | 是 | 每日签到、连续奖励、累计奖励、全勤补足 | 统一下月末到期 |
| 周末日卡 | 否 | 周六/周日自然日使用权 | 当天 00:00 到次日 00:00 |
| 保护卡 | 否 | 漏签时保护连续天数 | 每个活动周期最多使用 1 张 |
| 徽章/成就 | 否 | 低成本正反馈 | 不涉及额度 |

明确不再使用：

- 永久账户余额。
- 三日/七日单独到期额度。
- 周末额外现金额度。

### 5.2 签到额度

签到额度总预算约 `$20/月/用户`，全部计入 `monthly_checkin_credit_cap_per_user`。

| 奖励项 | 金额 | 说明 |
| --- | --- | --- |
| 每日签到 | `$0.25` | 每天一次，周末也发 |
| 首次参与 | `$1` | 首次参与活动时发，计入月度封顶 |
| 连续 3 天 | `$0.50` | 只发一次 |
| 连续 7 天 | `$1` + 1 张保护卡 | 保护卡不补发漏签奖励 |
| 连续 14 天 | `$1.50` | 只发一次 |
| 连续 21 天 | `$2` | 只发一次 |
| 累计 10 天 | `$1` | 只发一次 |
| 累计 20 天 | `$2` | 只发一次 |
| 累计 25 天 | `$2` | 只发一次 |
| 全勤 | 补足到 `$20` | 不超过月度封顶 |

规则：

- 如果用户未全勤，按实际达到的每日、连续、累计奖励发放。
- 如果用户已接近 `$20` 封顶，后续金额奖励只发到剩余额度。
- 达到 `$20` 后，后续签到仍计入连续天数、累计天数和日卡权益，但不再增加签到额度。
- 签到额度不可提现、不可转赠、不可兑换现金。

### 5.3 周六日卡

周六签到奖励：

- 周六整天日卡 1 张。
- 不额外发周六现金额度。
- 计入连续天数和累计天数。

日卡有效期：

- `starts_at`: 周六 00:00:00 `Asia/Shanghai`
- `expires_at`: 周日 00:00:00 `Asia/Shanghai`
- 用户如果在周六 20:00 才签到，仍然到周日 00:00 失效，不顺延到周日 20:00。

### 5.4 周日日卡

周日签到奖励：

- 周日整天日卡 1 张。
- 不额外发周日现金额度。
- 计入连续天数和累计天数。

日卡有效期：

- `starts_at`: 周日 00:00:00 `Asia/Shanghai`
- `expires_at`: 周一 00:00:00 `Asia/Shanghai`
- 不从点击签到时刻开始计算 24 小时。

### 5.5 周末日卡价值口径

周末日卡不进入额度钱包，只用于运营成本估算和页面权益说明。

建议估值：

| 权益 | 内部估值 | 说明 |
| --- | --- | --- |
| 周六日卡 | 约 `$2` | 使用专用活动订阅分组 |
| 周日日卡 | 约 `$2.50` | 可比周六稍高，强化周日参与 |
| 月度日卡权益 | 约 `$18` - `$22.50` | 取决于当月有 8、9 还是 10 个周末日 |

页面可展示：

```text
本月签到额度最高 $20，周末日卡权益约 $20
```

如果运营必须严格控制日卡价值在 `$20` 内，建议不要减少日卡数量，而是通过活动日卡分组的模型范围、每日请求额度或速率限制控制成本。

### 5.6 保护卡规则

- 连续 7 天时发 1 张保护卡。
- 用户漏签 1 天时，可消耗保护卡维持连续天数。
- 保护卡不补发漏签当天的签到额度和周末日卡。
- 每个活动周期最多使用 1 张。
- 使用保护卡后，页面标记为“已保护连续”，避免误解为补签成功。

## 6. 成本控制建议

### 6.1 成本边界

按单个全勤用户估算：

| 类型 | 月度预算 | 说明 |
| --- | --- | --- |
| 签到额度 | 约 `$20` | 钱包内限时额度，下月末到期 |
| 周末日卡 | 约 `$20` 感知价值 | 不进钱包，通过活动订阅分组控制成本 |
| 永久余额 | `$0` | 签到活动不发永久余额 |

关键原则：

- 签到额度必须有 `expires_at`，不可复用 `users.balance`。
- 周末日卡必须使用专用活动分组，不覆盖用户已有付费订阅。
- 页面展示时把 `$20` 签到额度和 `$20` 周末日卡分开。
- 后台必须设置单用户月度签到额度封顶、全站签到额度预算、周末日卡发放数量和实际成本看板。

### 6.2 三档预算

保守版：

- 签到额度封顶 `$10/月`
- 每日签到 `$0.15`
- 连续/累计奖励较小，全勤补足到 `$10`
- 周末日卡使用更严格的活动分组

平衡版：

- 签到额度封顶 `$15/月`
- 每日签到 `$0.20`
- 连续/累计奖励降低约 25%，全勤补足到 `$15`
- 周末日卡规则不变，通过活动分组限额控制成本

标准版：

- 签到额度封顶 `$20/月`
- 每日签到 `$0.25`
- 连续/累计奖励按首期推荐规则
- 周末日卡权益约 `$20/月`

推荐先上标准版。

### 6.3 预算熔断

建议增加活动级熔断参数：

- `monthly_checkin_credit_cap_per_user`: 单用户月度签到额度上限，标准版建议 `$20`。
- `monthly_weekend_day_card_value_per_user`: 单用户月度周末日卡估值，标准版约 `$20`。
- `campaign_checkin_credit_budget`: 全站签到额度预算。
- `campaign_weekend_day_card_budget`: 全站周末日卡成本预算或发放数量预算。
- `campaign_actual_cost_alert`: 按实际 API 成本触发告警。

当全站预算达到 80% 时，停止首次参与额外奖励和非必要运营加成；达到 100% 时，保留已承诺的每日签到和日卡规则，暂停新增活动奖励。

## 7. 页面方案

### 7.1 用户端页面结构

页面入口：

- 侧边栏保留“每日签到”。
- 首页 Dashboard 增加“今日未签到”提示。
- 周五晚上到周日可在签到入口显示“周末日卡”标记。

页面模块：

- 顶部：活动名称、今日状态、当前连续天数、累计签到天数。
- 本月权益：展示“签到额度最高 `$20`，下月末到期”和“周末日卡权益约 `$20`”。
- 今日奖励：显示今日可领签到额度、是否有周末日卡、到期时间。
- 月历：显示每日已签/未签/周末/里程碑。
- 额度进度：已领签到额度、剩余可领额度、到期日期。
- 周末日卡：展示本周六、周日奖励和固定失效时间。
- 里程碑进度：连续 3/7/14/21/全勤。
- 累计保底进度：累计 10/20/25 天。
- 最近记录：展示签到额度和日卡发放明细。
- 规则说明：时区、下月末到期、日卡自然日有效期、断签和保护卡。

成功反馈：

- 签到成功后弹出奖励明细：
  - 今日签到额度已到账。
  - 当前月累计已领签到额度。
  - 签到额度到期日期。
  - 如为周末，显示日卡有效期。
  - 距离下个里程碑还差几天。

### 7.2 管理端配置

建议新增“签到活动”配置页：

- 活动开关。
- 活动名称。
- 活动开始/结束时间。
- 时区。
- 签到额度上限。
- 签到额度到期策略：下月末到期。
- 每日签到额度。
- 连续奖励配置。
- 累计奖励配置。
- 全勤补足规则。
- 周六/周日日卡分组 ID。
- 周末日卡模型范围、每日限额、速率限制。
- 保护卡规则。
- 人群限制。
- 全站预算和熔断阈值。

## 8. 后端设计

### 8.1 当前系统基础

当前已经有：

- `user_checkins`：每日签到记录。
- `CheckinService`：按北京时间计算今天、连续天数、写入签到。
- `users.balance`：可加减账户余额。
- `user_subscriptions`：用户订阅，有 `starts_at`、`expires_at`。
- `subscription_plans` / `groups`：可表达日卡/订阅权益。

当前不足：

- 签到奖励是硬编码 `$0.02`，并写入永久余额。
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
    daily_checkin_credit DECIMAL(20, 10) NOT NULL DEFAULT 0,
    monthly_checkin_credit_cap_per_user DECIMAL(20, 10) NOT NULL DEFAULT 0,
    credit_expiry_policy VARCHAR(40) NOT NULL DEFAULT 'end_of_next_month',
    saturday_group_id BIGINT,
    sunday_group_id BIGINT,
    monthly_weekend_day_card_value_per_user DECIMAL(20, 10) NOT NULL DEFAULT 0,
    campaign_checkin_credit_budget DECIMAL(20, 10),
    campaign_weekend_day_card_budget DECIMAL(20, 10),
    milestone_rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    catchup_rules JSONB NOT NULL DEFAULT '{}'::jsonb,
    protection_rules JSONB NOT NULL DEFAULT '{}'::jsonb,
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

新增到期签到额度账本：

```sql
CREATE TABLE user_expiring_credits (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    source_type VARCHAR(30) NOT NULL,
    source_id BIGINT NOT NULL,
    campaign_id BIGINT REFERENCES checkin_campaigns(id),
    amount DECIMAL(20, 10) NOT NULL,
    remaining_amount DECIMAL(20, 10) NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 8.3 下月末到期计算

输入：

- `local_date`
- `timezone = Asia/Shanghai`

计算：

```text
credit_month_start = first day of local_date month 00:00:00
expiry_local = credit_month_start + 2 months
expires_at = expiry_local converted to UTC
```

示例：

```text
local_date = 2026-08-12
expires_at = 2026-10-01 00:00:00 Asia/Shanghai
用户侧显示：2026-09-30 23:59 到期
```

### 8.4 周末日卡发放逻辑

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
- 如果 `now >= expires_at`，不允许补发该日卡。
- 日卡使用专用活动分组，避免覆盖用户已有付费订阅。
- 如果用户已有同一专用分组活动订阅，只允许把 `expires_at` 更新为 `max(existing.expires_at, campaign_day_end)`，不允许缩短已有订阅。

### 8.5 API 建议

用户端：

- `GET /api/v1/user/check-in/campaign/status`
  - 返回活动信息、今日奖励、月度额度进度、额度到期日、日历、里程碑、周末日卡信息。
- `POST /api/v1/user/check-in`
  - 保留现有路径，但返回签到额度、额度到期日、日卡发放明细。
- `GET /api/v1/user/check-in/history`
  - 返回签到记录和奖励明细。

管理端：

- `GET /api/v1/admin/check-in/campaigns`
- `POST /api/v1/admin/check-in/campaigns`
- `PUT /api/v1/admin/check-in/campaigns/:id`
- `POST /api/v1/admin/check-in/campaigns/:id/publish`
- `GET /api/v1/admin/check-in/campaigns/:id/stats`

### 8.6 幂等和并发

必须保证：

- 同一用户同一天只能签到一次。
- 同一奖励只能发放一次。
- 用户重复点击、网络重试不会重复发额度或日卡。
- 签到记录、额度发放、日卡发放在同一事务内完成。

关键约束：

- `user_checkins (user_id, checkin_date)` 唯一。
- `checkin_reward_grants (campaign_id, user_id, reward_key)` 唯一。

## 9. 风控规则

建议首版加入：

- 只有正常状态用户可参与。
- 新注册用户可参与，但周末日卡可设置注册满 24 小时后可领。
- 同一用户每天一次，按账号维度，不按 IP。
- 对异常批量注册、同 IP 多账号、短时间批量领取做后台报表。
- 签到额度不可转赠、不可提现、不可兑换现金。
- 管理员可按用户撤销活动奖励。
- 到期额度应优先于永久余额或充值余额消耗；如果系统没有永久余额参与，本条可简化为“优先消耗最早到期额度”。

## 10. 指标看板

用户指标：

- 每日签到人数。
- 签到转化率：访问签到页后点击签到的比例。
- 连续 3/7/14/21/全勤人数。
- 累计 10/20/25 天达成人数。
- 周六/周日签到人数。
- 断签后回流人数。

成本指标：

- 发放签到额度总额。
- 签到额度实际消耗额。
- 签到额度过期未使用额。
- 单用户 `$20` 月度封顶触达率。
- 周末日卡发放数。
- 周末日卡产生的 API 成本。

业务指标：

- 活动参与用户 7 日留存。
- 活动参与用户周末 API 请求数。
- 活动后付费转化。
- 日卡体验后购买订阅的转化率。

## 11. 上线步骤

### 阶段 1：签到额度账本

- 把当前固定 `$0.02` 永久余额改为到期签到额度。
- 增加 `user_expiring_credits`。
- 实现下月末到期计算。
- 增加单用户月度 `$20` 签到额度封顶。
- 页面展示今日到账、累计已领、到期时间。

### 阶段 2：月度活动

- 增加 `checkin_campaigns`。
- 增加月历视图。
- 支持连续/累计里程碑。
- 支持全勤补足到 `$20`。
- 支持活动开关。

### 阶段 3：周末日卡

- 新增专用“周六签到日卡”和“周日签到日卡”订阅分组。
- 新增固定自然日发放逻辑。
- 增加日卡发放幂等记录。
- 页面展示周末日卡固定失效时间。

### 阶段 4：计费扣减

- 修改计费扣减优先级，优先消耗最早到期的签到额度。
- 到期额度自动不可用。
- 在账单/用量明细里区分充值额度、签到额度和日卡权益。

### 阶段 5：风控和运营看板

- 管理端活动统计。
- 奖励发放记录查询。
- 异常领取报表。
- 周末日卡实际 API 成本看板。

## 12. 推荐首期配置

活动名：

```text
月度签到挑战
```

活动周期：

```text
2026-08-01 00:00:00 Asia/Shanghai
到
2026-09-01 00:00:00 Asia/Shanghai
```

签到额度：

```text
月度签到额度封顶：$20
额度类型：限时活动额度，不发永久额度
额度到期：领取月份的下月末到期
每日签到：$0.25
首次参与：$1，计入 $20 封顶
连续 3 天：$0.50
连续 7 天：$1 + 1 张保护卡
连续 14 天：$1.50
连续 21 天：$2
累计 10 天：$1
累计 20 天：$2
累计 25 天：$2
全勤：补足到 $20 + 全勤徽章
```

周末日卡：

```text
周六：周六整天日卡，有效期周六 00:00 到周日 00:00
周日：周日整天日卡，有效期周日 00:00 到周一 00:00
日卡不折算进钱包额度
月度日卡权益感知价值：约 $20
```

预算封顶：

```text
单用户月度签到额度封顶：$20
单用户月度永久额度封顶：$0
单用户月度周末日卡权益估值：约 $20
全站预算达到 80%：停止首次参与额外奖励和非必要运营加成
全站预算达到 100%：保留已承诺每日签到和日卡规则，暂停新增活动奖励
```

## 13. 待确认问题

- 签到额度是否直接新增 `user_expiring_credits` 账本，还是先用活动专用订阅分组承载？
- 周末日卡要使用现有某个订阅分组，还是创建专用活动分组？
- 周末日卡是否不限量，还是按活动分组设置日限额/模型范围？
- `$20` 周末日卡价值是否只作为运营估值，不在页面展示精确折算金额？
- 保护卡每月是否固定 1 张？是否允许后台给特定用户补发？
- 首期活动从 2026-08-01 开始，还是尽快从当前日期启动？
