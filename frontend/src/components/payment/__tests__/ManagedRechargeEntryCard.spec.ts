import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import ManagedRechargeEntryCard from '../ManagedRechargeEntryCard.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  messages: {
    zh: {
      payment: {
        memberRecharge: {
          imageAlt: 'GPT Plus 和 Pro 订阅卡片',
          title: '订阅 GPT Plus-Pro',
          description: '选择需要的会员套餐',
          plus: 'GPT Plus',
          pro: 'GPT Pro',
          featurePlan: 'Plus / Pro 套餐可选',
          featureProgress: '订单进度随时查看',
          featureRefund: '失败后退款或人工核对',
          subscribeNow: '立即订阅',
        },
      },
    },
  },
})

describe('ManagedRechargeEntryCard', () => {
  it('renders the subscription image and plan information', () => {
    const wrapper = mount(ManagedRechargeEntryCard, { global: { plugins: [i18n] } })

    expect(wrapper.get('img').attributes('alt')).toBe('GPT Plus 和 Pro 订阅卡片')
    expect(wrapper.text()).toContain('订阅 GPT Plus-Pro')
    expect(wrapper.text()).toContain('GPT Plus')
    expect(wrapper.text()).toContain('GPT Pro')
    expect(wrapper.text()).toContain('订单进度随时查看')
  })

  it('emits select when the subscribe button is clicked', async () => {
    const wrapper = mount(ManagedRechargeEntryCard, { global: { plugins: [i18n] } })

    await wrapper.get('[data-testid="managed-recharge-subscribe-button"]').trigger('click')

    expect(wrapper.emitted('select')).toHaveLength(1)
  })
})
