import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'

const here = dirname(fileURLToPath(import.meta.url))
const read = (path: string) => readFileSync(resolve(here, path), 'utf8')

describe('Account Contributions admin integration surface', () => {
  it('registers an administrator-only route and visible sidebar entry', () => {
    const router = read('../../../router/index.ts')
    const route = router.slice(
      router.indexOf("path: '/admin/account-contributions'"),
      router.indexOf("path: '/admin/announcements'"),
    )
    expect(route).toContain('requiresAuth: true')
    expect(route).toContain('requiresAdmin: true')

    const sidebar = read('../../../components/layout/AppSidebar.vue')
    expect(sidebar).toContain("path: '/admin/account-contributions'")
    expect(sidebar).not.toContain('flagAccountContribution')
  })

  it('keeps the page read-only and avoids credential fields', () => {
    const page = read('../AccountContributionsView.vue')
    const api = read('../../../api/admin/accountContributions.ts')
    expect(page).toContain('accountContributionsAPI.getOverview()')
    expect(page).not.toContain('type="password"')
    expect(api).not.toContain('encrypted_payload')
    expect(api).not.toContain('upstream_identity_hash')
  })

  it('keeps Chinese and English locale trees symmetric', () => {
    expect(Object.keys(zh.admin.accountContributions)).toEqual(Object.keys(en.admin.accountContributions))
    expect(zh.nav.accountContributions).toBeTruthy()
    expect(en.nav.accountContributions).toBeTruthy()
  })
})
