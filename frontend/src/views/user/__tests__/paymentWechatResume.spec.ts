import { describe, expect, it } from 'vitest'
import type { SubscriptionAddonProduct } from '@/types/payment'
import { parseWechatResumeRoute, stripWechatResumeQuery } from '../paymentWechatResume'

describe('parseWechatResumeRoute', () => {
  it('prefers the opaque resume token over legacy openid query params', () => {
    expect(parseWechatResumeRoute({
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-123',
      openid: 'openid-123',
      payment_type: 'wxpay',
      amount: '12.5',
      order_type: 'subscription',
      plan_id: '7',
    }, [], [], 88)).toEqual({
      wechatResumeToken: 'resume-token-123',
      paymentType: 'wxpay',
      orderType: 'subscription',
      orderAmount: 0,
      planId: 7,
      addonProductId: undefined,
      subscriptionId: undefined,
    })
  })

  it('falls back to legacy openid-based resume when opaque token is absent', () => {
    expect(parseWechatResumeRoute({
      wechat_resume: '1',
      openid: 'openid-123',
      payment_type: 'wxpay',
      amount: '12.5',
      order_type: 'balance',
    }, [], [], 88)).toEqual({
      openid: 'openid-123',
      paymentType: 'wxpay',
      orderType: 'balance',
      orderAmount: 12.5,
      planId: undefined,
      addonProductId: undefined,
      subscriptionId: undefined,
    })
  })

  it('restores an add-on purchase target and price', () => {
    const product: SubscriptionAddonProduct = {
      id: 5,
      sku: 'addon-usd-100',
      name: '100 USD add-on',
      quota_usd: 100,
      price: 23.99,
      for_sale: true,
      sort_order: 40,
    }
    expect(parseWechatResumeRoute({
      wechat_resume: '1',
      openid: 'openid-123',
      payment_type: 'wxpay',
      order_type: 'addon',
      addon_product_id: '5',
      subscription_id: '19',
    }, [], [product], 88)).toEqual({
      openid: 'openid-123',
      paymentType: 'wxpay',
      orderType: 'addon',
      orderAmount: 23.99,
      planId: undefined,
      addonProductId: 5,
      subscriptionId: 19,
    })
  })
})

describe('stripWechatResumeQuery', () => {
  it('removes both opaque-token and legacy resume params from the route query', () => {
    expect(stripWechatResumeQuery({
      foo: 'bar',
      wechat_resume: '1',
      wechat_resume_token: 'resume-token-123',
      openid: 'openid-123',
      payment_type: 'wxpay',
      amount: '12.5',
      order_type: 'subscription',
      plan_id: '7',
      addon_product_id: '5',
      subscription_id: '19',
      state: 'state-123',
      scope: 'snsapi_base',
    })).toEqual({
      foo: 'bar',
    })
  })
})
