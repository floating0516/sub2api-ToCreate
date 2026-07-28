import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const viewPath = resolve(dirname(fileURLToPath(import.meta.url)), '../CustomPageView.vue')
const viewSource = readFileSync(viewPath, 'utf8')

describe('CustomPageView Quick Start access', () => {
  it('gates both the wizard and direct custom-page lookup', () => {
    expect(viewSource).toContain('v-if="showQuickStartWizard"')
    expect(viewSource).toContain('FeatureFlags.quickStartInstaller')
    expect(viewSource).toContain(
      'isQuickStartPage.value && !canAccessQuickStartInstaller.value',
    )
  })
})
