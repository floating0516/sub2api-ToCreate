import { afterEach, describe, expect, it, vi } from 'vitest'
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
          shopSubtitle: '选择商品并在当前页面完成下单',
          reloadShop: '重新加载小店',
          openShop: '新窗口打开',
          iframeTitle: 'GPT Plus / Pro 订阅小店',
        },
      },
    },
  },
})

describe('LdxpShopEmbed', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('embeds the configured Chain Shop and provides a new-window fallback', async () => {
    const wrapper = mount(LdxpShopEmbed, {
      global: {
        plugins: [i18n],
        stubs: {
          Icon: true,
        },
      },
    })

    const frame = wrapper.get('[data-testid="ldxp-shop-frame"]')
    const externalLink = wrapper.get('a[target="_blank"]')

    expect(frame.attributes('src')).toBe('https://pay.ldxp.cn/shop/FIDK51J9')
    expect(externalLink.attributes('href')).toBe('https://pay.ldxp.cn/shop/FIDK51J9')
    expect(frame.attributes('referrerpolicy')).toBe('strict-origin-when-cross-origin')

    await frame.trigger('load')
    expect(wrapper.find('[data-testid="ldxp-shop-loading"]').exists()).toBe(false)
  })

  it('reveals the iframe when the upstream page does not emit a load event', async () => {
    vi.useFakeTimers()
    const wrapper = mount(LdxpShopEmbed, {
      global: {
        plugins: [i18n],
        stubs: {
          Icon: true,
        },
      },
    })

    expect(wrapper.find('[data-testid="ldxp-shop-loading"]').exists()).toBe(true)

    await vi.advanceTimersByTimeAsync(7000)

    expect(wrapper.find('[data-testid="ldxp-shop-loading"]').exists()).toBe(false)
  })
})
