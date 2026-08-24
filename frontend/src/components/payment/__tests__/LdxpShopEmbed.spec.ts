import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import LdxpShopEmbed from '../LdxpShopEmbed.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  messages: {
    zh: {
      payment: {
        memberRecharge: {
          shopTitle: '订阅 GPT Plus / Pro',
          shopSubtitle: '商品选择与支付将在链动小铺完成',
          openShop: '前往链动小铺',
        },
      },
    },
  },
})

describe('LdxpShopEmbed', () => {
  it('opens the configured Chain Shop in a top-level window', () => {
    const wrapper = mount(LdxpShopEmbed, {
      global: {
        plugins: [i18n],
        stubs: {
          Icon: true,
        },
      },
    })

    const externalLink = wrapper.get('[data-testid="ldxp-shop-link"]')

    expect(externalLink.attributes('href')).toBe('https://pay.ldxp.cn/shop/ToCreate')
    expect(externalLink.attributes('target')).toBe('_blank')
    expect(externalLink.attributes('rel')).toBe('noopener noreferrer')
    expect(wrapper.find('iframe').exists()).toBe(false)
  })
})
