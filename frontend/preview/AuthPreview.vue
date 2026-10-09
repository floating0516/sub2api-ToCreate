<template>
  <section class="auth-preview" :data-step="step" :data-mode="mode" :data-busy="busy" aria-labelledby="auth-title">
    <div ref="shape" class="auth-shape" :class="{ 'auth-shape--complete': complete }">
      <div ref="inside" class="auth-inside">
        <div class="auth-topline">
          <div class="auth-mark" :class="{ 'is-busy': busy, 'is-complete': complete }" aria-hidden="true">
            <svg viewBox="0 0 32 32" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round">
              <circle class="auth-ring" cx="16" cy="16" r="12" />
              <path ref="markPath" :d="mark" />
            </svg>
          </div>
          <span class="auth-kicker">{{ t('home.authPreview.demo') }}</span>
        </div>
        <div v-show="step === 'form'" ref="tabs" class="auth-tabs" role="tablist" :aria-label="t('home.authPreview.choose')" @keydown="tabKey">
          <span ref="indicator" class="auth-indicator" aria-hidden="true" />
          <button id="login-tab" type="button" role="tab" :aria-selected="mode === 'login'" :tabindex="mode === 'login' ? 0 : -1" aria-controls="auth-panel" @click="choose('login')">{{ t('home.authPreview.login') }}</button>
          <button id="register-tab" type="button" role="tab" :aria-selected="mode === 'register'" :tabindex="mode === 'register' ? 0 : -1" aria-controls="auth-panel" @click="choose('register')">{{ t('home.authPreview.register') }}</button>
        </div>
        <Transition name="auth-content" mode="out-in" @before-leave="beforeLeave" @enter="enterPanel" @after-enter="afterEnter">
          <div :key="contentKey" id="auth-panel" ref="panel" :role="step === 'form' ? 'tabpanel' : undefined" :aria-labelledby="step === 'form' ? `${mode}-tab` : 'auth-title'">
            <header class="auth-heading">
              <h1 id="auth-title" ref="heading" tabindex="-1">{{ title }}</h1>
              <p>{{ subtitle }}</p>
            </header>
            <template v-if="complete">
              <button type="button" class="entry-link auth-submit" data-action="restart" @click="restart"><span>{{ t('home.authPreview.again') }}</span><span class="entry-arrow"><Icon name="arrowRight" :stroke-width="1.6" aria-hidden="true" /></span></button>
              <a class="auth-home-link" :href="homeHref" @click.prevent="emit('home')">{{ t('home.authPreview.home') }}</a>
            </template>
            <form v-else novalidate :aria-busy="busy" @submit.prevent="submit">
              <template v-if="step === 'verify'">
                <div class="auth-field">
                  <label for="auth-code">{{ t('home.authPreview.code') }}</label>
                  <input id="auth-code" v-model="code" class="auth-code" type="text" inputmode="numeric" autocomplete="off" maxlength="6" placeholder="000000" :readonly="busy" :aria-invalid="Boolean(errors.code)" :aria-describedby="errors.code ? 'code-error' : 'code-hint'" @input="errors.code = ''" />
                  <p v-if="errors.code" id="code-error" class="auth-error" role="alert">{{ t(errors.code) }}</p>
                  <p v-else id="code-hint" class="auth-hint">{{ t('home.authPreview.codeHint') }}</p>
                </div>
              </template>
              <template v-else>
                <div class="auth-field">
                  <label for="auth-email">{{ t('home.authPreview.email') }}</label>
                  <input id="auth-email" v-model="email" type="email" inputmode="email" autocomplete="off" autocapitalize="none" spellcheck="false" placeholder="you@example.com" :readonly="busy" :aria-invalid="Boolean(errors.email)" :aria-describedby="errors.email ? 'email-error' : undefined" @input="errors.email = ''" />
                  <p v-if="errors.email" id="email-error" class="auth-error" role="alert">{{ t(errors.email) }}</p>
                </div>
                <template v-if="step === 'form'">
                  <div class="auth-field">
                    <label for="auth-password">{{ t('home.authPreview.password') }}</label>
                    <div class="auth-password-wrap">
                      <input id="auth-password" v-model="password" :type="showPassword ? 'text' : 'password'" autocomplete="off" :readonly="busy" :placeholder="t('home.authPreview.passwordHint')" :aria-invalid="Boolean(errors.password)" :aria-describedby="errors.password ? 'password-error' : undefined" @input="errors.password = ''" />
                      <button type="button" class="auth-reveal" :aria-label="t(showPassword ? 'home.authPreview.hide' : 'home.authPreview.show')" :aria-pressed="showPassword" @click="showPassword = !showPassword"><Icon :name="showPassword ? 'eyeOff' : 'eye'" :stroke-width="1.6" aria-hidden="true" /></button>
                    </div>
                    <p v-if="errors.password" id="password-error" class="auth-error" role="alert">{{ t(errors.password) }}</p>
                  </div>
                  <div v-if="mode === 'register'" class="auth-field">
                    <label for="auth-confirm">{{ t('home.authPreview.confirm') }}</label>
                    <input id="auth-confirm" v-model="confirmation" :type="showPassword ? 'text' : 'password'" autocomplete="off" :readonly="busy" :placeholder="t('home.authPreview.confirmHint')" :aria-invalid="Boolean(errors.confirmation)" :aria-describedby="errors.confirmation ? 'confirm-error' : undefined" @input="errors.confirmation = ''" />
                    <p v-if="errors.confirmation" id="confirm-error" class="auth-error" role="alert">{{ t(errors.confirmation) }}</p>
                  </div>
                </template>
              </template>
              <div class="auth-form-tools">
                <button type="button" data-action="fill" :disabled="busy" @click="fillExample">{{ t('home.authPreview.fill') }}</button>
                <button v-if="step === 'form' && mode === 'login'" type="button" data-action="forgot" @click="navigate('forgot')">{{ t('home.authPreview.forgot') }}</button>
                <button v-else-if="step !== 'form'" type="button" data-action="back" @click="navigate('form')">{{ t('home.authPreview.back') }}</button>
              </div>
              <button type="submit" class="entry-link auth-submit" :class="{ 'is-busy': busy }" :aria-disabled="busy || switching" data-action="submit">
                <span aria-live="polite">{{ submitLabel }}</span>
                <span class="entry-arrow" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path class="auth-submit-arrow" d="M5 12h14m-6-6 6 6-6 6"/><circle class="auth-submit-ring" cx="12" cy="12" r="7" /></svg></span>
              </button>
              <button v-if="busy" type="button" class="auth-cancel" data-action="cancel" @click="cancelRequest">{{ t('home.authPreview.cancel') }}</button>
            </form>
          </div>
        </Transition>
      </div>
    </div>
    <p class="auth-demo-note">{{ t('home.authPreview.notice') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../src/components/icons/Icon.vue'
import { indicatorFrames, springFrames } from '../src/components/home/modelGatewayMotion'

type Mode = 'login' | 'register'
type Step = 'form' | 'verify' | 'forgot' | 'done' | 'sent'
const props = defineProps<{ mode: Mode; homeHref: string }>()
const emit = defineEmits<{ mode: [value: Mode]; home: [] }>()
const { t, locale } = useI18n()
const step = ref<Step>('form'), busy = ref(false), switching = ref(false)
const email = ref(''), password = ref(''), confirmation = ref(''), code = ref(''), showPassword = ref(false)
const errors = reactive({ email: '', password: '', confirmation: '', code: '' })
const shape = ref<HTMLElement>(), inside = ref<HTMLElement>(), panel = ref<HTMLElement>(), heading = ref<HTMLElement>()
const tabs = ref<HTMLElement>(), indicator = ref<HTMLElement>(), markPath = ref<SVGPathElement>()
const complete = computed(() => step.value === 'done' || step.value === 'sent')
const contentKey = computed(() => step.value === 'form' ? props.mode : step.value)
const title = computed(() => t(`home.authPreview.${step.value === 'form' ? (props.mode === 'login' ? 'welcome' : 'create') : step.value + 'Title'}`))
const subtitle = computed(() => step.value === 'verify' ? t('home.authPreview.verifySubtitle', { email: email.value }) : t(`home.authPreview.${step.value === 'form' ? props.mode : step.value}Subtitle`))
const submitLabel = computed(() => t(`home.authPreview.${busy.value ? 'working' : step.value === 'verify' ? 'verifyAction' : step.value === 'forgot' ? 'send' : props.mode}`))
const marks = {
  login: 'M11 15 L11 11 Q11 6 16 6 Q21 6 21 11 L21 15 M10 15 L22 15 L22 25 L10 25 Z',
  register: 'M11 15 L11 11 Q11 6 16 6 Q21 6 21 11 L24 11 M10 15 L22 15 L22 25 L10 25 Z',
  mail: 'M8 12 L16 18 Q16 18 16 18 Q16 18 16 18 L24 12 M7 10 L25 10 L25 23 L7 23 Z',
  done: 'M9 16 L14 21 Q14 21 14 21 Q14 21 14 21 L24 10 M9 16 L14 21 L24 10 L14 21 Z'
}
const mark = computed(() => complete.value ? marks.done : step.value === 'verify' || step.value === 'forgot' ? marks.mail : marks[props.mode])
let disposed = false, epoch = 0, timer: ReturnType<typeof setTimeout> | undefined
let resizeObserver: ResizeObserver | undefined, motionQuery: MediaQueryList | undefined
let heightAnimation: Animation | undefined, tabAnimation: Animation | undefined, markAnimation: Animation | undefined
let knownHeight = 0
const reduced = () => motionQuery?.matches || document.hidden
function clearErrors() { for (const key of Object.keys(errors) as (keyof typeof errors)[]) errors[key] = '' }
function cancelRequest() { epoch++; if (timer) clearTimeout(timer); timer = undefined; busy.value = false }
function choose(value: Mode) {
  if (switching.value || value === props.mode) return
  cancelRequest(); clearErrors(); emit('mode', value)
}
function navigate(value: Step) { if (switching.value) return; cancelRequest(); clearErrors(); step.value = value }
function restart() { password.value = ''; confirmation.value = ''; code.value = ''; navigate('form') }
function fillExample() {
  if (busy.value) return
  if (step.value === 'verify') code.value = '123456'
  else { email.value = 'hello@example.com'; password.value = 'DemoOnly2026!'; confirmation.value = password.value }
  clearErrors()
}
async function submit() {
  if (busy.value || switching.value) return
  clearErrors()
  if (step.value === 'verify') {
    if (code.value !== '123456') errors.code = 'home.authPreview.invalidCode'
  } else {
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value.trim())) errors.email = 'home.authPreview.invalidEmail'
    if (step.value === 'form') {
      if (password.value.length < 8) errors.password = 'home.authPreview.invalidPassword'
      if (props.mode === 'register' && confirmation.value !== password.value) errors.confirmation = 'home.authPreview.mismatch'
    }
  }
  const first = Object.entries(errors).find(([,value]) => value)
  if (first) {
    await nextTick()
    panel.value?.querySelector<HTMLInputElement>(`#auth-${first[0] === 'confirmation' ? 'confirm' : first[0]}`)?.focus({ preventScroll: true })
    return
  }
  busy.value = true
  const token = ++epoch
  timer = setTimeout(() => {
    if (disposed || token !== epoch) return
    timer = undefined; busy.value = false
    if (step.value === 'forgot') step.value = 'sent'
    else if (step.value === 'form' && props.mode === 'register') step.value = 'verify'
    else step.value = 'done'
    password.value = ''; confirmation.value = ''
  }, 1050)
}
function tabKey(event: KeyboardEvent) {
  if (!['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) return
  event.preventDefault()
  if (switching.value) return
  const target: Mode = event.key === 'Home' ? 'login' : event.key === 'End' ? 'register' : props.mode === 'login' ? 'register' : 'login'
  choose(target)
  tabs.value?.querySelector<HTMLButtonElement>(`#${target}-tab`)?.focus({ preventScroll: true })
}
function beforeLeave(el: Element) {
  switching.value = true
  const node = el as HTMLElement
  // Keep focus in the shared tabs for tab switches; otherwise park it on the region.
  if (node.contains(document.activeElement)) { shape.value?.focus({ preventScroll: true }) }
  node.inert = true
}
function afterEnter() {
  switching.value = false
  resizeShape()
  if (!tabs.value?.contains(document.activeElement)) heading.value?.focus({ preventScroll: true })
}
function enterPanel() { resizeShape(true) }
function resizeShape(force = false) {
  if (switching.value && !force) return
  if (!shape.value || !inside.value || disposed) return
  const target = Math.ceil(inside.value.getBoundingClientRect().height) + 2
  if (target === knownHeight) return
  const from = shape.value.getBoundingClientRect().height
  const initialized = knownHeight > 0
  knownHeight = target
  heightAnimation?.cancel()
  shape.value.style.height = `${target}px`
  if (initialized && !reduced()) heightAnimation = shape.value.animate(springFrames({height:from},{height:target}),{duration:400,easing:'linear'})
}
function moveIndicator() {
  if (!tabs.value || !indicator.value || step.value !== 'form') return
  const selected = tabs.value.querySelector<HTMLElement>(`#${props.mode}-tab`)
  if (!selected) return
  const current = indicator.value.getBoundingClientRect(), parent = tabs.value.getBoundingClientRect()
  const left = selected.offsetLeft, right = left + selected.offsetWidth
  tabAnimation?.cancel()
  indicator.value.style.left = `${left}px`; indicator.value.style.width = `${right-left}px`
  if (!reduced()) tabAnimation = indicator.value.animate(indicatorFrames(current.left-parent.left,current.right-parent.left,left,right),{duration:380,easing:'linear'})
}
function finishMotion() {
  for (const animation of [heightAnimation, tabAnimation, markAnimation]) {
    if (animation && animation.playState !== 'idle') animation.finish()
  }
  shape.value?.classList.toggle('is-hidden', document.hidden)
}
watch(() => props.mode, () => { cancelRequest(); clearErrors(); step.value = 'form'; showPassword.value = false; confirmation.value = ''; void nextTick(moveIndicator) })
watch(step, () => { void nextTick(moveIndicator) })
watch(locale, () => { void nextTick(resizeShape) })
watch(mark, (to, from) => {
  markAnimation?.cancel()
  if (markPath.value && !reduced()) markAnimation = markPath.value.animate([{d:`path("${from}")`},{d:`path("${to}")`}],{duration:360,easing:'cubic-bezier(.2,.8,.2,1)'})
}, {flush:'post'})
onMounted(() => {
  shape.value?.setAttribute('tabindex','-1')
  motionQuery = matchMedia('(prefers-reduced-motion: reduce)')
  motionQuery.addEventListener('change',finishMotion)
  document.addEventListener('visibilitychange',finishMotion)
  resizeShape(); moveIndicator()
  resizeObserver = new ResizeObserver(() => { resizeShape(); moveIndicator() })
  if (inside.value) resizeObserver.observe(inside.value)
})
onBeforeUnmount(() => {
  disposed = true; cancelRequest(); resizeObserver?.disconnect()
  heightAnimation?.cancel(); tabAnimation?.cancel(); markAnimation?.cancel()
  motionQuery?.removeEventListener('change',finishMotion)
  document.removeEventListener('visibilitychange',finishMotion)
})
</script>
