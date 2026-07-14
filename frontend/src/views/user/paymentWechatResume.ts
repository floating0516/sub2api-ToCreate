import type { LocationQuery, LocationQueryRaw } from 'vue-router'
import type { SubscriptionAddonProduct, SubscriptionPlan } from '@/types/payment'
import { normalizeVisibleMethod } from '@/components/payment/paymentFlow'

export interface ParsedWechatResumeRoute {
  orderAmount: number
  orderType: 'balance' | 'subscription' | 'addon'
  paymentType: string
  planId?: number
  addonProductId?: number
  subscriptionId?: number
  openid?: string
  wechatResumeToken?: string
}

function readQueryString(query: LocationQuery, key: string): string {
  const value = query[key]
  if (Array.isArray(value)) {
    return typeof value[0] === 'string' ? value[0] : ''
  }
  return typeof value === 'string' ? value : ''
}

export function hasWechatResumeQuery(query: LocationQuery): boolean {
  if (readQueryString(query, 'wechat_resume') === '1') {
    return true
  }
  return readQueryString(query, 'wechat_resume_token') !== ''
    || readQueryString(query, 'openid') !== ''
}

export function parseWechatResumeRoute(
  query: LocationQuery,
  plans: SubscriptionPlan[],
  addonProducts: SubscriptionAddonProduct[],
  fallbackBalanceAmount: number,
): ParsedWechatResumeRoute | null {
  if (!hasWechatResumeQuery(query)) {
    return null
  }

  const wechatResumeToken = readQueryString(query, 'wechat_resume_token')
  const paymentType = normalizeVisibleMethod(readQueryString(query, 'payment_type')) || 'wxpay'
  const planId = Number.parseInt(readQueryString(query, 'plan_id'), 10)
  const hasPlanId = Number.isFinite(planId) && planId > 0
  const addonProductId = Number.parseInt(readQueryString(query, 'addon_product_id'), 10)
  const subscriptionId = Number.parseInt(readQueryString(query, 'subscription_id'), 10)
  const hasAddonProductId = Number.isFinite(addonProductId) && addonProductId > 0
  const hasSubscriptionId = Number.isFinite(subscriptionId) && subscriptionId > 0
  const rawOrderType = readQueryString(query, 'order_type')
  const orderType = rawOrderType === 'addon' || (hasAddonProductId && hasSubscriptionId)
    ? 'addon'
    : rawOrderType === 'subscription' || hasPlanId
      ? 'subscription'
      : 'balance'

  if (wechatResumeToken) {
    return {
      wechatResumeToken,
      paymentType,
      orderType,
      orderAmount: 0,
      planId: hasPlanId ? planId : undefined,
      addonProductId: hasAddonProductId ? addonProductId : undefined,
      subscriptionId: hasSubscriptionId ? subscriptionId : undefined,
    }
  }

  const openid = readQueryString(query, 'openid')
  if (!openid) {
    return null
  }

  const rawAmount = Number.parseFloat(readQueryString(query, 'amount'))
  const orderAmount = Number.isFinite(rawAmount) && rawAmount > 0
    ? rawAmount
    : (orderType === 'subscription'
      ? (plans.find(plan => plan.id === planId)?.price ?? 0)
      : orderType === 'addon'
        ? (addonProducts.find(product => product.id === addonProductId)?.price ?? 0)
        : fallbackBalanceAmount)

  return {
    openid,
    paymentType,
    orderType,
    orderAmount,
    planId: hasPlanId ? planId : undefined,
    addonProductId: hasAddonProductId ? addonProductId : undefined,
    subscriptionId: hasSubscriptionId ? subscriptionId : undefined,
  }
}

export function stripWechatResumeQuery(query: LocationQuery): LocationQueryRaw {
  const nextQuery: LocationQueryRaw = { ...query }
  delete nextQuery.wechat_resume
  delete nextQuery.wechat_resume_token
  delete nextQuery.openid
  delete nextQuery.state
  delete nextQuery.scope
  delete nextQuery.payment_type
  delete nextQuery.amount
  delete nextQuery.order_type
  delete nextQuery.plan_id
  delete nextQuery.addon_product_id
  delete nextQuery.subscription_id
  return nextQuery
}
