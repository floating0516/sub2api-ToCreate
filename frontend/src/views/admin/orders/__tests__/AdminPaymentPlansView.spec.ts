import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import AdminPaymentPlansView from '../AdminPaymentPlansView.vue'

const {
  getAllGroups,
  getConfig,
  getPlans,
  getAddonProducts,
  updateConfig,
  updatePlan,
  deletePlan,
  updateAddonProduct,
} = vi.hoisted(() => ({
  getAllGroups: vi.fn(),
  getConfig: vi.fn(),
  getPlans: vi.fn(),
  getAddonProducts: vi.fn(),
  updateConfig: vi.fn(),
  updatePlan: vi.fn(),
  deletePlan: vi.fn(),
  updateAddonProduct: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }),
}))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: {
    getConfig,
    getPlans,
    getAddonProducts,
    updateConfig,
    updatePlan,
    deletePlan,
    updateAddonProduct,
  },
}))

vi.mock('@/api/admin', () => ({
  default: { groups: { getAll: getAllGroups } },
}))

const DataTableStub = {
  props: ['columns', 'data', 'loading'],
  template: `
    <div data-testid="data-table">
      <span data-testid="column-keys">{{ columns.map(column => column.key).join(',') }}</span>
      <div v-for="row in data" :key="row.id">
        <slot name="cell-unit_price" :row="row" :value="row.unit_price" />
      </div>
    </div>
  `,
}

function mountView() {
  return mount(AdminPaymentPlansView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        DataTable: DataTableStub,
        ConfirmDialog: true,
        GroupBadge: true,
        Icon: true,
        PlanEditDialog: true,
        AddonProductEditDialog: true,
      },
    },
  })
}

describe('AdminPaymentPlansView product catalog', () => {
  beforeEach(() => {
    getAllGroups.mockReset().mockResolvedValue([])
    getConfig.mockReset().mockResolvedValue({ data: { addon_purchase_enabled: false, subscription_usd_to_cny_rate: 0 } })
    getPlans.mockReset().mockResolvedValue({ data: [] })
    getAddonProducts.mockReset().mockResolvedValue({
      data: [{
        id: 1,
        sku: 'addon-usd-10',
        name: '10 美元加油包',
        quota_usd: 10,
        price: 1.99,
        original_price: null,
        for_sale: true,
        sort_order: 10,
      }],
    })
    updateConfig.mockReset()
    updatePlan.mockReset()
    deletePlan.mockReset()
    updateAddonProduct.mockReset()
  })

  it('renders segmented catalog tabs and loads add-on products on demand', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="plans-catalog"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="addons-catalog"]').exists()).toBe(false)

    await wrapper.get('[data-testid="catalog-tab-addons"]').trigger('click')
    await flushPromises()

    expect(getAddonProducts).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-testid="addons-catalog"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="column-keys"]').text()).toContain('sku,name,quota_usd,price,unit_price')
    expect(wrapper.text()).toContain('¥0.1990')
    expect(wrapper.text()).toContain('$0.1990')
  })
})
