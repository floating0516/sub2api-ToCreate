import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AccountsView.vue')
const componentSource = readFileSync(componentPath, 'utf8')

describe('AccountsView layout', () => {
  it('uses the official table layout instead of account cards', () => {
    expect(componentSource).toContain('<DataTable')
    expect(componentSource).not.toContain('<AccountCard')
    expect(componentSource).not.toContain('<AccountCardList')
    expect(componentSource).not.toContain('content-variant="cards"')
  })

  it('links API key account names only through a sanitized upstream URL', () => {
    expect(componentSource).toContain('v-if="accountHomepageUrl(row)"')
    expect(componentSource).toContain('target="_blank"')
    expect(componentSource).toContain('rel="noopener noreferrer"')
    expect(componentSource).toContain("row.type !== 'apikey'")
    expect(componentSource).toContain('sanitizeUrl(row.credentials.base_url)')
  })
})
