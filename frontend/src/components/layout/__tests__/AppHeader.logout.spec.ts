import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AppHeader from '@/components/layout/AppHeader.vue'

const {
  appStore,
  authStore,
  logoutMock,
  onboardingStore,
  replaceMock,
} = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: { custom_menu_items: [] },
    contactInfo: '',
    docUrl: '',
    toggleMobileSidebar: vi.fn(),
  },
  authStore: {
    isAdmin: false,
    isSimpleMode: false,
    logout: vi.fn(),
    user: {
      balance: 0,
      email: 'user@example.com',
      role: 'user',
      username: 'Test User',
    },
  },
  logoutMock: vi.fn(),
  onboardingStore: { replay: vi.fn() },
  replaceMock: vi.fn(),
}))

authStore.logout = logoutMock

vi.mock('vue-router', () => ({
  useRoute: () => ({ meta: {}, name: 'Dashboard', params: {} }),
  useRouter: () => ({ replace: replaceMock }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
  useOnboardingStore: () => onboardingStore,
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] }),
}))

vi.mock('@/utils/featureFlags', () => ({
  FeatureFlags: { modelPlaza: 'model_plaza' },
  isFeatureFlagEnabled: () => false,
}))

function mountHeader() {
  return mount(AppHeader, {
    global: {
      stubs: {
        AnnouncementBell: true,
        Icon: true,
        LocaleSwitcher: true,
        RouterLink: RouterLinkStub,
        SubscriptionProgressMini: true,
      },
    },
  })
}

async function clickLogout() {
  const wrapper = mountHeader()
  await wrapper.get('button[aria-label="common.userMenu"]').trigger('click')
  await wrapper.get('[data-testid="app-header-logout"]').trigger('click')
  await flushPromises()
}

describe('AppHeader logout navigation', () => {
  beforeEach(() => {
    logoutMock.mockReset()
    replaceMock.mockReset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('returns to the unified home authentication entry after logout', async () => {
    logoutMock.mockResolvedValue(undefined)

    await clickLogout()

    expect(logoutMock).toHaveBeenCalledOnce()
    expect(replaceMock).toHaveBeenCalledWith('/home')
  })

  it('still returns home when the logout request rejects', async () => {
    logoutMock.mockRejectedValue(new Error('network unavailable'))
    vi.spyOn(console, 'error').mockImplementation(() => undefined)

    await clickLogout()

    expect(replaceMock).toHaveBeenCalledWith('/home')
  })
})
