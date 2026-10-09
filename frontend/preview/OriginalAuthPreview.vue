<template>
  <div class="tc-auth-shell original-auth" :class="{ 'original-auth--still': still, 'original-auth--hidden': hidden }" :data-step="step" :data-busy="busy" @pointerup.capture="releaseButton" @keyup.capture="releaseKey">
    <a :href="homeHref" class="tc-auth-home" @click.prevent="emit('home')"><Icon name="arrowLeft" size="sm" /><span>{{ t('home.redesign.backHome') }}</span></a>
    <div class="original-preview-tools">
      <span :title="t('home.authPreview.notice')">{{ t('home.authPreview.demo') }}</span>
      <button type="button" data-action="fill" :disabled="busy || changing" @click="fillExample">{{ t('home.authPreview.fill') }}</button>
      <button type="button" :aria-label="t('home.simplePreview.language')" @click="switchLanguage">{{ locale === 'zh' ? 'EN' : '中文' }}</button>
    </div>
    <div class="tc-auth-frame">
      <section class="tc-auth-intro">
        <div class="tc-auth-brand"><img :src="logo" alt="" /><span>ToCreate</span></div>
        <div class="tc-auth-copy"><p>{{ t('home.redesign.authEyebrow') }}</p><h1>{{ t('home.redesign.authHeadline') }}</h1><span>{{ t('home.redesign.authDescription') }}</span></div>
        <div class="tc-auth-preview" aria-hidden="true"><p>{{ t('home.redesign.diagramTitle') }}</p><ul><li><span class="tc-auth-provider tc-auth-claude">C</span>Claude</li><li><span class="tc-auth-provider tc-auth-gpt">O</span>GPT</li><li><span class="tc-auth-provider tc-auth-gemini">G</span>Gemini</li></ul></div>
      </section>
      <section class="tc-auth-content">
        <div class="tc-auth-mobile-brand"><img :src="logo" alt="" /><span>ToCreate</span></div>
        <div class="tc-auth-form">
          <div class="email-auth-layer is-embedded">
            <section ref="dialog" class="email-auth-dialog" aria-labelledby="email-auth-title" tabindex="-1">
              <button v-if="step !== 'email'" type="button" class="email-auth-icon-button email-auth-back" data-action="back" :aria-label="t('home.authPreview.back')" @click="back"><Icon name="arrowLeft" size="sm" /></button>
              <div class="email-auth-brand" aria-hidden="true"><span class="email-auth-orbit email-auth-orbit-one" /><span class="email-auth-orbit email-auth-orbit-two" /><img :src="logo" alt="" /></div>
              <div ref="windowEl" class="original-stage-window">
                <Transition name="original-step" mode="out-in" @before-leave="beforeLeave" @before-enter="beforeEnter" @enter="enter" @after-enter="afterEnter">
                  <div :key="step" ref="stage" class="email-auth-stage" :aria-busy="busy">
                    <header v-if="step !== 'done' && step !== 'sent'" class="email-auth-copy">
                      <p>{{ eyebrow }}</p><h2 id="email-auth-title" tabindex="-1">{{ title }}</h2>
                      <span v-if="step === 'login' || step === 'register'" class="email-auth-account" :title="email">{{ email }}</span>
                      <span v-else>{{ subtitle }}</span>
                    </header>
                    <div v-if="step === 'done' || step === 'sent'" class="email-auth-progress email-auth-success">
                      <div class="email-auth-success-mark" aria-hidden="true"><Icon name="check" size="lg" /></div>
                      <p>{{ t('auth.emailFirst.ready') }}</p><h2 id="email-auth-title" tabindex="-1">{{ t('home.authPreview.doneTitle') }}</h2>
                      <span>{{ t(step === 'sent' ? 'home.authPreview.sentSubtitle' : 'home.authPreview.doneSubtitle') }}</span>
                      <button class="email-auth-text-button" type="button" data-action="restart" @click="go('email')">{{ t('home.authPreview.again') }}</button>
                    </div>
                    <form v-else novalidate @submit.prevent="submit">
                      <template v-if="step === 'email' || step === 'forgot'">
                        <label class="email-auth-label" for="email-auth-email">{{ t('auth.emailLabel') }}</label>
                        <div class="email-auth-input-shell" :class="{ 'is-error': error }"><Icon name="mail" size="sm" /><input id="email-auth-email" v-model="email" type="email" autocomplete="off" inputmode="email" autocapitalize="none" spellcheck="false" :placeholder="t('auth.emailPlaceholder')" :readonly="busy" :aria-invalid="Boolean(error)" aria-describedby="auth-message" @input="error = ''" /></div>
                      </template>
                      <template v-else-if="step === 'login' || step === 'register'">
                        <label class="email-auth-label" for="email-auth-password">{{ t('auth.passwordLabel') }}</label>
                        <div class="email-auth-input-shell" :class="{ 'is-error': error }"><Icon name="lock" size="sm" /><input id="email-auth-password" v-model="password" :type="showPassword ? 'text' : 'password'" autocomplete="off" :placeholder="t(step === 'login' ? 'auth.passwordPlaceholder' : 'auth.createPasswordPlaceholder')" :readonly="busy" :aria-invalid="Boolean(error)" aria-describedby="auth-message" @input="error = ''" /><button class="email-auth-field-action" type="button" :aria-label="t(showPassword ? 'auth.emailFirst.hidePassword' : 'auth.emailFirst.showPassword')" :aria-pressed="showPassword" @click="showPassword = !showPassword"><Icon :name="showPassword ? 'eyeOff' : 'eye'" size="sm" /></button></div>
                        <template v-if="step === 'register'">
                          <label class="email-auth-label email-auth-confirm-label" for="email-auth-confirm-password">{{ t('auth.confirmPassword') }}</label>
                          <div class="email-auth-input-shell" :class="{ 'is-error': error }"><Icon name="shield" size="sm" /><input id="email-auth-confirm-password" v-model="confirmation" :type="showPassword ? 'text' : 'password'" autocomplete="off" :placeholder="t('auth.confirmPasswordPlaceholder')" :readonly="busy" :aria-invalid="Boolean(error)" aria-describedby="auth-message" @input="error = ''" /></div>
                        </template>
                      </template>
                      <template v-else-if="step === 'verify'">
                        <label class="email-auth-label" for="email-auth-code">{{ t('auth.verificationCode') }}</label>
                        <div class="email-auth-otp" :class="{ 'is-error': error }">
                          <input id="email-auth-code" v-model="code" class="email-auth-otp-input" type="text" inputmode="numeric" autocomplete="off" maxlength="6" :readonly="busy" :aria-invalid="Boolean(error)" aria-describedby="auth-message" @input="error = ''" />
                          <span v-for="index in 6" :key="index" :class="{ 'is-filled': code[index - 1], 'is-current': code.length === index - 1 }" aria-hidden="true">{{ code[index - 1] || '' }}</span>
                        </div>
                      </template>
                      <div class="email-auth-message-row">
                        <p id="auth-message" class="email-auth-message" :class="{ 'is-error': error }" role="status">{{ error ? t(error) : hint }}</p>
                        <button v-if="step === 'login'" class="email-auth-inline-link" type="button" data-action="forgot" @click="go('forgot')">{{ t('auth.forgotPassword') }}</button>
                      </div>
                      <div v-if="step === 'email'" class="email-auth-choice">
                        <button type="button" :class="mode === 'login' ? 'email-auth-primary' : 'email-auth-secondary'" data-action="login" @click="continueWith('login')">{{ t('auth.signIn') }}</button>
                        <button type="button" :class="mode === 'register' ? 'email-auth-primary' : 'email-auth-secondary'" data-action="register" @click="continueWith('register')">{{ t('auth.createAccount') }}</button>
                      </div>
                      <button v-else type="submit" class="email-auth-primary" data-action="submit" :aria-disabled="busy || changing">
                        <span class="original-button-copy" :class="{ 'is-busy': busy }"><span class="email-auth-spinner" aria-hidden="true" /><span aria-live="polite">{{ buttonText }}</span></span>
                      </button>
                    </form>
                    <div v-if="step === 'login' || step === 'register'" class="email-auth-switch">
                      <span>{{ t(step === 'login' ? 'auth.dontHaveAccount' : 'auth.alreadyHaveAccount') }}</span><button type="button" data-action="switch" @click="switchMode">{{ t(step === 'login' ? 'auth.createAccount' : 'auth.signIn') }}</button>
                    </div>
                    <button v-if="busy" class="email-auth-text-button original-cancel" type="button" data-action="cancel" @click="cancel">{{ t('home.authPreview.cancel') }}</button>
                  </div>
                </Transition>
              </div>
              <p class="email-auth-legal">{{ t('home.authPreview.notice') }}</p>
            </section>
          </div>
        </div>
        <div class="tc-auth-footer" />
        <p class="tc-auth-copyright">© {{ new Date().getFullYear() }} ToCreate. {{ t('home.footer.allRightsReserved') }}</p>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '../src/components/icons/Icon.vue'
import { springAt, springFrames } from '../src/components/home/modelGatewayMotion'
import logo from './logo.png'
import './original-auth.css'
import './original-auth-motion.css'

type Step = 'email' | 'login' | 'register' | 'verify' | 'forgot' | 'done' | 'sent'
type Mode = 'login' | 'register'
const props = defineProps<{ mode: Mode; homeHref: string }>()
const emit = defineEmits<{ mode: [value: Mode]; home: [] }>()
const { t, locale } = useI18n()
const step = ref<Step>('email'), busy = ref(false), changing = ref(false), hidden = ref(document.hidden)
const still = new URLSearchParams(location.search).has('still')
const email = ref(''), password = ref(''), confirmation = ref(''), code = ref(''), error = ref(''), showPassword = ref(false)
const stage = ref<HTMLElement>(), dialog = ref<HTMLElement>(), windowEl = ref<HTMLElement>()
const eyebrow = computed(() => t(step.value === 'email' ? 'auth.emailFirst.welcome' : step.value === 'login' ? 'auth.emailFirst.signInEyebrow' : step.value === 'register' ? 'auth.emailFirst.createEyebrow' : 'auth.emailFirst.verifyEyebrow', {siteName:'ToCreate'}))
const title = computed(() => t(step.value === 'email' ? 'auth.emailFirst.title' : step.value === 'login' ? 'auth.welcomeBack' : step.value === 'register' ? 'auth.createAccount' : step.value === 'verify' ? 'home.authPreview.verifyTitle' : 'home.authPreview.forgotTitle'))
const subtitle = computed(() => step.value === 'verify' ? t('home.authPreview.verifySubtitle', {email:email.value}) : t(step.value === 'forgot' ? 'home.authPreview.forgotSubtitle' : 'auth.emailFirst.emailDescription'))
const hint = computed(() => t(step.value === 'email' || step.value === 'forgot' ? 'auth.emailFirst.emailHint' : step.value === 'login' ? 'auth.emailFirst.passwordHint' : step.value === 'verify' ? 'home.authPreview.codeHint' : 'auth.passwordHint'))
const buttonText = computed(() => t(busy.value ? 'home.authPreview.working' : step.value === 'login' ? 'auth.signIn' : step.value === 'register' ? 'auth.emailFirst.sendCode' : step.value === 'verify' ? 'auth.emailFirst.verifyCode' : 'home.authPreview.send'))
let timer: ReturnType<typeof setTimeout> | undefined, epoch = 0, disposed = false, knownHeight = 0
let observer: ResizeObserver | undefined, query: MediaQueryList | undefined, heightMotion: Animation | undefined
const animations = new Set<Animation>()
const reduced = () => still || query?.matches || hidden.value
function cancel() { epoch++; if (timer) clearTimeout(timer); timer = undefined; busy.value = false }
function go(to: Step) { if (changing.value) return; cancel(); error.value = ''; showPassword.value = false; step.value = to }
function back() { go(step.value === 'verify' ? 'register' : 'email') }
function continueWith(mode: Mode) {
  if (changing.value || busy.value) return
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value.trim())) { error.value = 'auth.invalidEmail'; focusField('email-auth-email'); return }
  emit('mode',mode); go(mode)
}
function switchMode() { const mode = step.value === 'login' ? 'register' : 'login'; if (changing.value) return; emit('mode',mode); go(mode) }
function fillExample() { if (busy.value || changing.value) return; email.value = 'hello@example.com'; password.value = 'DemoOnly2026!'; confirmation.value = password.value; code.value = '123456'; error.value = '' }
function focusField(id: string) { void nextTick(() => dialog.value?.querySelector<HTMLElement>(`#${id}`)?.focus({preventScroll:true})) }
function submit() {
  if (busy.value || changing.value) return
  if (step.value === 'email') { continueWith(props.mode); return }
  error.value = ''
  if ((step.value === 'login' || step.value === 'register') && password.value.length < 6) { error.value = 'auth.passwordMinLength'; focusField('email-auth-password'); return }
  if (step.value === 'register' && password.value !== confirmation.value) { error.value = 'home.authPreview.mismatch'; focusField('email-auth-confirm-password'); return }
  if (step.value === 'verify' && code.value !== '123456') { error.value = 'home.authPreview.invalidCode'; focusField('email-auth-code'); return }
  if (step.value === 'forgot' && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value.trim())) { error.value = 'auth.invalidEmail'; focusField('email-auth-email'); return }
  const token = ++epoch; busy.value = true
  timer = setTimeout(() => { if (disposed || token !== epoch) return; timer = undefined; busy.value = false; step.value = step.value === 'register' ? 'verify' : step.value === 'forgot' ? 'sent' : 'done'; password.value = ''; confirmation.value = ''; code.value = '' }, 900)
}
function beforeLeave(el: Element) { changing.value = true; if (el.contains(document.activeElement)) dialog.value?.focus({preventScroll:true}); (el as HTMLElement).inert = true }
function beforeEnter(el: Element) { (el as HTMLElement).inert = true }
function enter() { void nextTick(() => resize(true)) }
function afterEnter() { changing.value = false; if (stage.value) { stage.value.inert = false; observer?.disconnect(); observer?.observe(stage.value) } resize(); focusField('email-auth-title') }
function resize(force = false) {
  if (disposed || !windowEl.value || !stage.value || (changing.value && !force)) return
  const height = Math.ceil(stage.value.getBoundingClientRect().height)
  if (height === knownHeight) return
  const before = windowEl.value.getBoundingClientRect().height, initialized = knownHeight > 0
  knownHeight = height; heightMotion?.cancel(); windowEl.value.style.height = `${height}px`
  if (initialized && !reduced()) heightMotion = windowEl.value.animate(springFrames({height:before},{height}),{duration:340,easing:'linear'})
}
function releaseButton(event: Event) {
  if (reduced()) return
  const button = (event.target as Element).closest<HTMLButtonElement>('.email-auth-primary, .email-auth-secondary')
  if (!button || button.disabled || button.getAttribute('aria-disabled') === 'true') return
  const motion = button.animate(Array.from({length:25},(_,i) => { const p = springAt(i/24); return {offset:i/24,transform:`translateY(${1-p}px) scale(${.985+.015*p})`} }),{duration:220,easing:'linear'})
  animations.add(motion); motion.onfinish = () => animations.delete(motion)
}
function releaseKey(event: KeyboardEvent) { if (event.key === 'Enter' || event.key === ' ') releaseButton(event) }
function visibility() { hidden.value = document.hidden; if (reduced()) { heightMotion?.cancel(); for (const animation of animations) animation.cancel(); animations.clear() } }
function switchLanguage() { locale.value = locale.value === 'zh' ? 'en' : 'zh'; document.documentElement.lang = locale.value; const q = new URLSearchParams(location.search); q.set('lang',locale.value); history.replaceState(null,'',`?${q}`) }
watch(() => props.mode, value => { if ((step.value === 'login' || step.value === 'register') && value !== step.value) go(value) })
watch(locale, () => { void nextTick(resize) })
onMounted(() => { query = matchMedia('(prefers-reduced-motion: reduce)'); query.addEventListener('change',visibility); document.addEventListener('visibilitychange',visibility); resize(); observer = new ResizeObserver(() => resize()); if (stage.value) observer.observe(stage.value) })
onBeforeUnmount(() => { disposed = true; cancel(); observer?.disconnect(); heightMotion?.cancel(); for (const animation of animations) animation.cancel(); animations.clear(); query?.removeEventListener('change',visibility); document.removeEventListener('visibilitychange',visibility) })
</script>
