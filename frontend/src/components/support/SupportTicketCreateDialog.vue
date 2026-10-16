<template>
  <BaseDialog
    :show="show"
    :title="t('support.createTicket')"
    width="normal"
    :close-on-escape="!submitting && !uploadBusy"
    :show-close-button="!submitting && !uploadBusy"
    @close="requestClose('close')"
  >
    <p v-if="!submitTicket" role="status" class="mb-5 rounded-lg bg-amber-50 p-3 text-sm leading-6 text-amber-800 dark:bg-amber-950/30 dark:text-amber-200">{{ t('support.directCreateUnavailable') }}</p>
    <form id="support-create-form" novalidate @submit.prevent="submit">
      <fieldset :disabled="submitting" class="space-y-4">
        <div>
          <label for="support-create-type" class="label">{{ t('support.type') }}</label>
          <Select id="support-create-type" v-model="draft.type" :options="ticketTypes" :disabled="submitting" :aria-label="t('support.type')" />
        </div>
        <div>
          <label for="support-create-title" class="label">{{ t('support.formTitle') }}</label>
          <input id="support-create-title" ref="titleInput" v-model="draft.title" class="input" :placeholder="t('support.formTitlePlaceholder')" :aria-invalid="attempted && !validTitle" :aria-describedby="attempted && !validTitle ? 'support-title-error' : undefined" />
          <p v-if="attempted && !validTitle" id="support-title-error" class="mt-1 text-sm text-red-600 dark:text-red-300">{{ t('support.validation.titleLength') }}</p>
        </div>
        <div>
          <label for="support-create-description" class="label">{{ t('support.formDescription') }}</label>
          <textarea id="support-create-description" ref="descriptionInput" v-model="draft.description" class="input min-h-28" :placeholder="t('support.formDescriptionPlaceholder')" :aria-invalid="attempted && !validDescription" :aria-describedby="attempted && !validDescription ? 'support-description-error' : undefined" />
          <p v-if="attempted && !validDescription" id="support-description-error" class="mt-1 text-sm text-red-600 dark:text-red-300">{{ t('support.validation.descriptionLength') }}</p>
        </div>
        <div>
          <label for="support-create-product" class="label">{{ t('support.product') }}</label>
          <input id="support-create-product" v-model="draft.product" class="input" :aria-invalid="attempted && !validProduct" />
          <p v-if="attempted && !validProduct" class="mt-1 text-sm text-red-600 dark:text-red-300">{{ t('support.validation.productLength') }}</p>
        </div>
        <div>
          <label for="support-create-order" class="label">{{ t('support.relatedOrder') }}</label>
          <Select id="support-create-order" v-model="draft.orderId" :options="orderOptions" :disabled="!submitTicket || ordersState !== 'ready' || submitting" searchable :aria-label="t('support.relatedOrder')" />
          <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">{{ t('support.managedOrdersOnly') }}</p>
          <p v-if="ordersState === 'loading'" role="status" class="mt-2 text-xs text-gray-500">{{ t('common.loading') }}</p>
          <div v-else-if="ordersState === 'error'" class="mt-2 flex flex-wrap items-center gap-2 text-xs text-red-600 dark:text-red-300"><span role="alert">{{ t('support.errors.loadOrders') }}</span><button type="button" class="btn btn-ghost btn-sm" :aria-label="t('common.refresh')" @click="emit('retryOrders')"><Icon name="refresh" size="sm" /></button></div>
          <p v-if="attempted && draft.type === 'order' && !draft.orderId" class="mt-1 text-sm text-red-600 dark:text-red-300">{{ t('support.validation.orderRequired') }}</p>
        </div>
        <SupportAttachmentPicker :upload-file="uploadFile" :disabled="submitting" :reset-key="attachmentReset" @change="draft.attachmentIds = $event" @pending="attachmentsPending = $event" @busy="uploadBusy = $event" />
      </fieldset>
    </form>
    <p v-if="error" role="alert" class="mt-4 text-sm text-red-600 dark:text-red-300">{{ t('support.errors.createTicket') }}</p>
    <div v-if="pendingExit" role="alert" class="mt-5 border-t border-gray-200 pt-4 dark:border-dark-700">
      <p class="text-sm text-gray-700 dark:text-gray-200">{{ t('support.unsavedDraft') }}</p>
    </div>
    <template #footer>
      <div class="flex w-full flex-wrap justify-end gap-3">
        <template v-if="pendingExit">
          <button type="button" class="btn btn-secondary" @click="pendingExit = null">{{ t('support.keepEditing') }}</button>
          <button type="button" class="btn btn-danger" @click="discardAndExit">{{ t('support.discardDraft') }}</button>
        </template>
        <template v-else>
          <button type="button" class="btn btn-secondary" :disabled="submitting || uploadBusy" @click="requestClose('close')">{{ t('common.close') }}</button>
          <button v-if="!submitTicket" type="button" class="btn btn-secondary" @click="requestClose('consultation')"><Icon name="chat" size="sm" class="mr-2" />{{ t('support.aiConsultation') }}</button>
          <button type="submit" form="support-create-form" data-testid="submit-ticket" class="btn btn-primary" :disabled="!submitTicket || submitting || attachmentsPending"><Icon name="plus" size="sm" class="mr-2" />{{ t(submitting ? 'support.processing' : 'support.createTicket') }}</button>
        </template>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import SupportAttachmentPicker from './SupportAttachmentPicker.vue'
import type { SupportAttachment } from '@/api/support'
import type { TicketFormValues } from './ticketPresentation'

const props = defineProps<{
  show: boolean
  submitTicket?: (values: TicketFormValues, requestId: string) => Promise<boolean>
  uploadFile?: (file: File) => Promise<SupportAttachment>
  orders?: { value: string; label: string }[]
  ordersState?: 'unavailable' | 'loading' | 'error' | 'ready'
}>()
const emit = defineEmits<{ close: []; consultation: []; submitted: []; retryOrders: [] }>()
const { t } = useI18n()
const emptyDraft = (): TicketFormValues => ({ type: 'pre-sale', title: '', description: '', product: 'tocreate', priority: 'normal', orderId: '', attachmentIds: [] })
const draft = reactive<TicketFormValues>(emptyDraft())
const attempted = ref(false)
const submitting = ref(false)
const error = ref(false)
const pendingExit = ref<'close' | 'consultation' | null>(null)
const titleInput = ref<HTMLInputElement | null>(null)
const descriptionInput = ref<HTMLTextAreaElement | null>(null)
const attachmentsPending = ref(false)
const uploadBusy = ref(false)
const attachmentReset = ref(0)
const dirty = computed(() => attachmentsPending.value || JSON.stringify(draft) !== JSON.stringify(emptyDraft()))
const validTitle = computed(() => { const length = Array.from(draft.title.trim()).length; return length >= 4 && length <= 200 })
const validDescription = computed(() => { const length = Array.from(draft.description.trim()).length; return length >= 10 && length <= 8000 })
const validProduct = computed(() => { const length = Array.from(draft.product.trim()).length; return length > 0 && length <= 128 })
const orderOptions = computed(() => [{ value: '', label: t('support.noRelatedOrder') }, ...(props.orders || [])])
let pendingRequest: { payload: string; id: string } | null = null
let disposed = false
const ticketTypes = computed(() => [
  { value: 'pre-sale', label: t('support.types.preSale') },
  { value: 'order', label: t('support.types.order') },
  { value: 'after-sale', label: t('support.types.afterSale') },
  { value: 'technical', label: t('support.types.technical') },
  { value: 'other', label: t('support.types.other') },
])

function reset() {
  Object.assign(draft, emptyDraft())
  attachmentReset.value++
  attachmentsPending.value = false
  uploadBusy.value = false
  pendingRequest = null
  attempted.value = false
  error.value = false
  pendingExit.value = null
}
function requestClose(destination: 'close' | 'consultation') {
  if (submitting.value || uploadBusy.value) return
  if (dirty.value) pendingExit.value = destination
  else { reset(); exit(destination) }
}
function exit(destination: 'close' | 'consultation') {
  if (destination === 'close') emit('close')
  else emit('consultation')
}
function discardAndExit() {
  const destination = pendingExit.value
  reset()
  if (destination) exit(destination)
}
async function submit() {
  if (!props.show || !props.submitTicket || submitting.value || pendingExit.value || attachmentsPending.value) return
  attempted.value = true
  if (!validTitle.value || !validDescription.value || !validProduct.value || draft.type === 'order' && !draft.orderId) {
    if (!validTitle.value) titleInput.value?.focus()
    else descriptionInput.value?.focus()
    return
  }
  submitting.value = true
  error.value = false
  try {
    const values = { ...draft, title: draft.title.trim(), description: draft.description.trim(), product: draft.product.trim(), attachmentIds: [...draft.attachmentIds] }
    const payload = JSON.stringify(values)
    if (pendingRequest?.payload !== payload) pendingRequest = { payload, id: crypto.randomUUID() }
    const completed = await props.submitTicket(values, pendingRequest.id)
    if (disposed) return
    if (completed === true) {
      reset()
      emit('submitted')
    } else error.value = true
  } catch {
    if (!disposed) error.value = true
  } finally {
    if (!disposed) submitting.value = false
  }
}
function beforeUnload(event: BeforeUnloadEvent) {
  if (!dirty.value) return
  event.preventDefault()
  event.returnValue = ''
}
onMounted(() => window.addEventListener('beforeunload', beforeUnload))
onUnmounted(() => {
  disposed = true
  window.removeEventListener('beforeunload', beforeUnload)
})
</script>
