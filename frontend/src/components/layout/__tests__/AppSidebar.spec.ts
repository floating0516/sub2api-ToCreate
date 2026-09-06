import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar scroll position persistence', () => {
  it('binds a template ref to the sidebar nav element', () => {
    expect(componentSource).toContain('ref="sidebarNavRef"')
    expect(componentSource).toContain('sidebar-nav')
  })

  it('declares sidebarNavRef in script setup', () => {
    expect(componentSource).toContain("const sidebarNavRef = ref<HTMLElement | null>(null)")
  })

  it('saves scroll position on beforeUnmount', () => {
    expect(componentSource).toContain('onBeforeUnmount')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('sidebarNavRef.value.scrollTop')
  })

  it('restores scroll position on mount', () => {
    expect(componentSource).toContain('onMounted')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('nextTick')
  })
})

describe('AppSidebar collapsible groups', () => {
  it('lets the user collapse a group even while a child route is active', () => {
    // The expand state must come from the user's override first, falling back
    // to the active-route heuristic only when the user has not clicked yet.
    expect(componentSource).toContain('const groupExpandOverrides = ref<Map<string, boolean>>(new Map())')
    expect(componentSource).not.toContain('expandedGroups.value.has(item.path) || isGroupActive(item)')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})

describe('AppSidebar Lihe Chat entry', () => {
  it('keeps API keys as the only sidebar entry point for Lihe imports', () => {
    expect(componentSource).not.toContain("{ path: '/integrations/lihe'")
    expect(componentSource).not.toContain("t('nav.liheChat')")
  })
})

describe('AppSidebar Quick Start access', () => {
  it('recognizes the deployed menu id and applies the installer feature gate', () => {
    expect(componentSource).toContain("id === 'codex-claude-import'")
    expect(componentSource).toContain('canAccessQuickStartInstaller()')
    expect(componentSource).toContain('FeatureFlags.quickStartInstaller')
  })
})

describe('AppSidebar user-page switches', () => {
  it('gates the redemption and order entries independently', () => {
    expect(componentSource).toContain('FeatureFlags.userRedeem')
    expect(componentSource).toContain('FeatureFlags.userOrders')
    expect(componentSource).toContain("path: '/redeem'")
    expect(componentSource).toContain('featureFlag: flagUserRedeem')
    expect(componentSource).toContain("path: '/orders'")
    expect(componentSource).toContain('featureFlag: flagUserOrders')
  })
})

describe('AppSidebar subscription entries', () => {
  it('hides managed recharge while keeping subscription management and user purchases', () => {
    expect(componentSource).not.toContain("{ path: '/admin/managed-recharge'")
    expect(componentSource).toContain("{ path: '/admin/subscriptions'")
    expect(componentSource).toContain("{ path: '/subscriptions'")
    expect(componentSource).toContain("{ path: '/purchase'")
  })
})
