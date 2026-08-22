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
          subscribeNow: '立即订阅',
          plans: {
            plus: { title: 'Plus', badge: 'PLUS', description: 'Plus 套餐' },
            pro5x: { title: 'Pro（5 倍）', badge: 'PRO 5X', description: '5 倍套餐' },
            pro20x: { title: 'Pro（20 倍）', badge: 'PRO 20X', description: '20 倍套餐' },
          },
        },
      },
    },
  },
})

describe('ManagedRechargeEntryCard', () => {
  it('renders the three membership plan cards', () => {
    const wrapper = mount(ManagedRechargeEntryCard, { global: { plugins: [i18n] } })

    expect(wrapper.findAll('[data-testid^="managed-recharge-plan-"]')).toHaveLength(3)
    expect(wrapper.text()).toContain('Plus')
    expect(wrapper.text()).toContain('Pro（5 倍）')
    expect(wrapper.text()).toContain('Pro（20 倍）')
    expect(wrapper.findAll('img')).toHaveLength(3)
  })

  it('emits the selected plan when a card is clicked', async () => {
    const wrapper = mount(ManagedRechargeEntryCard, { global: { plugins: [i18n] } })

    await wrapper.get('[data-testid="managed-recharge-plan-pro-20x"]').trigger('click')

    expect(wrapper.emitted('select')).toEqual([['pro-20x']])
  })
})
