<template>
  <section ref="root" class="mg-demo" :data-locale="locale" :data-state="state" :data-active="visible && pageVisible && !reduced" :aria-label="t('home.modelDemo.label')">
    <div class="mg-meta">
      <span><span class="mg-dot" />{{ t('home.modelDemo.label') }}</span>
      <button type="button" class="mg-reset" data-action="reset" :aria-label="t('home.modelDemo.reset')" @click="reset">
        <Icon name="refresh" :stroke-width="1.6" aria-hidden="true" />
      </button>
    </div>

    <div class="mg-stage">
      <!-- This element persists through every state. Only its content changes. -->
      <div ref="shape" class="mg-shape">
        <div ref="mark" class="mg-shared-mark" :class="{ 'is-loading': rendered === 'loading' }" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><circle ref="markRing" cx="8" cy="8" r="4" /><path ref="markPath" d="M11 11 L20 20 L20 16 M16 16 L19 13" /></svg></div>
        <div ref="content" class="mg-content" :class="{ 'mg-light-text': darkContent }">
          <template v-if="rendered === 'key'">
            <div class="mg-topline"><span class="mg-mark-slot" /><span>ToCreate</span><span class="mg-example">{{ t('home.modelDemo.sample') }}</span></div>
            <h3>{{ t('home.modelDemo.oneKey') }}</h3>
            <p class="mg-subtitle">Claude <span> / </span> GPT <span> / </span> Gemini</p>
            <div class="mg-key"><span>sk-demo</span><span aria-hidden="true">•••• •••• ••••</span></div>
            <button type="button" class="mg-primary" data-action="choose" data-focus @click="go('select')">{{ t('home.modelDemo.choose') }}<Icon name="arrowRight" :stroke-width="1.6" aria-hidden="true" /></button>
          </template>

          <template v-else-if="rendered === 'select'">
            <div class="mg-topline"><span class="mg-mark-slot" /><span class="mg-eyebrow">{{ t('home.modelDemo.sameKey') }}</span><span class="mg-step">01 / 03</span></div>
            <h3>{{ t('home.modelDemo.selectTitle') }}</h3>
            <div ref="tabs" class="mg-tabs" role="radiogroup" :aria-label="t('home.modelDemo.choose')" @keydown="onModelKey">
              <span ref="indicator" class="mg-indicator" aria-hidden="true" />
              <button v-for="(model, index) in models" :key="model" type="button" role="radio" :aria-checked="selected === index" :tabindex="selected === index ? 0 : -1" :data-model="index" :data-focus="selected === index ? '' : undefined" @click="select(index)">{{ model }}</button>
            </div>
            <div class="mg-model-detail"><span class="mg-small-label">{{ t('home.modelDemo.promptLabel') }}</span><p>{{ t('home.modelDemo.prompt') }}</p></div>
            <div class="mg-request-note"><Icon name="link" :stroke-width="1.6" aria-hidden="true" /><span>sk-demo · {{ models[selected] }}</span></div>
            <button type="button" class="mg-primary" data-action="request" @click="request">{{ t('home.modelDemo.try') }}<Icon name="arrowRight" :stroke-width="1.6" aria-hidden="true" /></button>
          </template>

          <template v-else-if="rendered === 'loading'">
            <div class="mg-loading-row"><span class="mg-mark-slot" /><div><h3>{{ t('home.modelDemo.loading') }}</h3><p class="mg-subtitle">{{ models[selected] }} · {{ t('home.modelDemo.simulated') }}</p></div></div>
            <button type="button" class="mg-text-action" data-action="cancel" data-focus @click="reset">{{ t('home.modelDemo.cancel') }}</button>
          </template>

          <template v-else-if="rendered === 'response'">
            <div class="mg-topline"><span class="mg-mark-slot" /><span>{{ models[selected] }}</span><span class="mg-step">02 / 03</span></div>
            <h3 tabindex="-1" data-focus>{{ t('home.modelDemo.responseTitle') }}</h3>
            <p class="mg-answer">{{ t(`home.modelDemo.responses.${selected}`) }}</p>
            <p class="mg-receipt"><Icon name="key" :stroke-width="1.6" aria-hidden="true" />{{ t('home.modelDemo.receipt') }}</p>
            <div class="mg-actions"><button type="button" class="mg-secondary" data-action="again" @click="go('select')">{{ t('home.modelDemo.again') }}</button><button type="button" class="mg-primary" data-action="done" @click="go('done')">{{ t('home.modelDemo.done') }}<Icon name="check" :stroke-width="1.6" aria-hidden="true" /></button></div>
          </template>

          <template v-else>
            <div class="mg-success-row"><span class="mg-mark-slot" /><div><h3>{{ t('home.modelDemo.success') }}</h3><p class="mg-subtitle">{{ t('home.modelDemo.successDetail') }}</p></div></div>
            <button type="button" class="mg-text-action" data-action="restart" data-focus @click="reset">{{ t('home.modelDemo.restart') }}<Icon name="arrowRight" :stroke-width="1.6" aria-hidden="true" /></button>
          </template>
        </div>
      </div>
    </div>
    <p class="mg-disclaimer">{{ t('home.modelDemo.disclaimer') }}</p>
    <p class="mg-sr-only" role="status" aria-live="polite" aria-atomic="true">{{ announcement }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { indicatorFrames, springFrames } from './modelGatewayMotion'

type State = 'key' | 'select' | 'loading' | 'response' | 'done'
const { t, locale } = useI18n()
const models = ['Claude', 'GPT', 'Gemini'] as const
const state = ref<State>('key'), rendered = ref<State>('key'), selected = ref(0)
const root = ref<HTMLElement>(), shape = ref<HTMLElement>(), content = ref<HTMLElement>()
const mark = ref<HTMLElement>(), markRing = ref<SVGCircleElement>(), markPath = ref<SVGPathElement>()
const tabs = ref<HTMLElement>(), indicator = ref<HTMLElement>()
const reduced = ref(false), visible = ref(true), pageVisible = ref(true)
const darkContent = computed(() => ['key', 'loading', 'done'].includes(rendered.value))
const announcement = computed(() => state.value === 'loading' ? t('home.modelDemo.loading') : state.value === 'response' ? t('home.modelDemo.responseTitle') : state.value === 'done' ? t('home.modelDemo.success') : '')
const allowed: Record<State, State[]> = { key: ['select'], select: ['loading'], loading: ['response'], response: ['select', 'done'], done: ['key'] }
let revision = 0, requestId = 0, timer: ReturnType<typeof setTimeout> | undefined, disposed = false
let observer: IntersectionObserver | undefined, resize: ResizeObserver | undefined, media: MediaQueryList | undefined
let morph: Animation | undefined, tabMotion: Animation | undefined
let markMotions: Animation[] = []
const animations = new Set<Animation>()
const motionAllowed = () => !reduced.value && visible.value && !document.hidden

function animate(el: Element, frames: ReturnType<typeof springFrames>, duration: number) {
  const animation = el.animate(frames, { duration, easing: 'linear', fill: 'none' })
  animations.add(animation)
  void animation.finished.catch(() => {}).finally(() => animations.delete(animation))
  return animation
}
async function finished(animation: Animation) { try { await animation.finished } catch { /* interrupted by a newer action */ } }
function settle() { for (const animation of animations) { try { animation.finish() } catch { animation.cancel() } } }
function clearRequest() { requestId++; if (timer !== undefined) clearTimeout(timer); timer = undefined }

function dimensions(target: State) {
  const available = Math.min(360, root.value?.clientWidth ?? 360)
  const width = target === 'loading' ? Math.min(280, available) : target === 'done' ? Math.min(320, available) : available
  const minHeight = { key: 244, select: 296, loading: 124, response: 296, done: 152 }[target]
  const radius = { key: 24, select: 24, loading: 36, response: 24, done: 30 }[target]
  return { width, minHeight, radius }
}

function sizeShape(target: State, animated: boolean) {
  const el = shape.value, inner = content.value
  if (!el || !inner) return
  const { width, minHeight, radius } = dimensions(target)
  const before = el.getBoundingClientRect(), css = getComputedStyle(el)
  const from = { width: before.width, height: before.height, borderRadius: parseFloat(css.borderRadius) }
  const fromColor = css.backgroundColor
  morph?.cancel()
  inner.style.width = `${width - 40}px`
  const height = Math.max(minHeight, inner.scrollHeight + 40)
  const to = { width, height, borderRadius: radius }
  const color = ['key', 'loading', 'done'].includes(target) ? '#191919' : '#fffefd'
  Object.assign(el.style, { width: `${width}px`, height: `${height}px`, borderRadius: `${radius}px`, backgroundColor: color })
  positionMark(target, height, animated)
  if (animated && motionAllowed()) {
    const frames = springFrames(from, to)
    // Color follows the same uninterrupted shape, without gradients or scaling.
    frames[0].backgroundColor = fromColor
    frames[frames.length - 1].backgroundColor = color
    morph = animate(el, frames, 380)
  }
}

// The same SVG and paths survive every state and move independently of fading text.
function positionMark(target: State, height: number, animated: boolean) {
  const el = mark.value, ring = markRing.value, path = markPath.value, inner = content.value
  const slot = inner?.querySelector<HTMLElement>('.mg-mark-slot')
  if (!el || !ring || !path || !slot || !inner) return
  const style = getComputedStyle(el), circle = getComputedStyle(ring), line = getComputedStyle(path)
  const from = { left: parseFloat(style.left) || 20, top: parseFloat(style.top) || 20 }
  const fromColor = style.color
  const previousRing = { cx: circle.getPropertyValue('cx'), cy: circle.getPropertyValue('cy'), r: circle.getPropertyValue('r'), strokeDasharray: circle.strokeDasharray, opacity: circle.opacity }
  const previousPath = { d: line.getPropertyValue('d'), opacity: line.opacity }
  for (const animation of markMotions) animation.cancel()
  const slotRect = slot.getBoundingClientRect(), innerRect = inner.getBoundingClientRect()
  const to = { left: 19 + slotRect.left - innerRect.left, top: (height - inner.offsetHeight) / 2 + slotRect.top - innerRect.top }
  const key = target === 'key' || target === 'select', loading = target === 'loading'
  const color = ['key', 'loading', 'done'].includes(target) ? '#f5f5f2' : '#363633'
  const nextRing = { cx: key ? '8px' : '12px', cy: key ? '8px' : '12px', r: key ? '4px' : '9px', strokeDasharray: loading ? '40 17' : '57 0', opacity: key || loading ? '1' : '0.22' }
  const nextPath = { d: key ? 'path("M11 11 L20 20 L20 16 M16 16 L19 13")' : 'path("M6 12 L10 16 L18 8 M18 8 L18 8")', opacity: loading ? '0' : '1' }
  Object.assign(el.style, { left: `${to.left}px`, top: `${to.top}px`, color })
  Object.assign(ring.style, nextRing); Object.assign(path.style, nextPath)
  if (animated && motionAllowed()) {
    const frames = springFrames(from, to)
    frames[0].color = fromColor; frames[frames.length - 1].color = color
    markMotions = [animate(el, frames, 380), animate(ring, [previousRing, nextRing], 380), animate(path, [previousPath, nextPath], 300)]
  }
}

async function present(target: State) {
  const token = ++revision
  const inner = content.value
  if (!inner) return
  const ownedFocus = inner.contains(document.activeElement)
  const oldFocus = document.activeElement
  for (const animation of inner.getAnimations()) animation.cancel()
  inner.inert = true
  if (motionAllowed()) await finished(animate(inner, [{ opacity: 1, filter: 'blur(0px)' }, { opacity: 0, filter: 'blur(2px)' }], 70))
  if (disposed || token !== revision) return
  inner.style.opacity = '0'
  rendered.value = target
  await nextTick()
  if (disposed || token !== revision) return
  sizeShape(target, true)
  if (target === 'select') positionIndicator(false)
  inner.style.opacity = '1'
  inner.inert = false
  if (motionAllowed()) animate(inner, [
    { opacity: 0, filter: 'blur(2px)', offset: 0 },
    { opacity: 0, filter: 'blur(2px)', offset: 0.32 },
    { opacity: 1, filter: 'blur(0px)', offset: 1 }
  ], 300)
  // Never steal focus back from navigation or the persistent reset control.
  if (ownedFocus && (document.activeElement === oldFocus || document.activeElement === document.body)) {
    inner.querySelector<HTMLElement>('[data-focus]')?.focus({ preventScroll: true })
  }
}

function go(target: State) {
  if (!allowed[state.value].includes(target)) return
  state.value = target
  void present(target)
}
function reset() {
  clearRequest()
  state.value = 'key'
  selected.value = 0
  void present('key')
}
function request() {
  if (state.value !== 'select') return
  clearRequest()
  const id = requestId
  go('loading')
  timer = setTimeout(() => {
    timer = undefined
    if (!disposed && id === requestId && state.value === 'loading') go('response')
  }, 1100)
}

function positionIndicator(animated: boolean) {
  const el = indicator.value, group = tabs.value
  const button = group?.querySelector<HTMLElement>(`[data-model="${selected.value}"]`)
  if (!el || !button || !group) return
  const current = el.getBoundingClientRect(), bounds = group.getBoundingClientRect()
  const left = current.left - bounds.left, right = current.right - bounds.left
  const nextLeft = button.offsetLeft, nextRight = nextLeft + button.offsetWidth
  tabMotion?.cancel()
  el.style.left = `${nextLeft}px`; el.style.width = `${button.offsetWidth}px`
  if (animated && motionAllowed()) tabMotion = animate(el, indicatorFrames(left, right, nextLeft, nextRight), 380)
}
function select(index: number) {
  if (state.value !== 'select' || selected.value === index) return
  selected.value = index
  void nextTick(() => positionIndicator(true))
}
function onModelKey(event: KeyboardEvent) {
  const moves: Record<string, number> = { ArrowRight: (selected.value + 1) % 3, ArrowDown: (selected.value + 1) % 3, ArrowLeft: (selected.value + 2) % 3, ArrowUp: (selected.value + 2) % 3, Home: 0, End: 2 }
  if (!(event.key in moves)) return
  event.preventDefault()
  const index = moves[event.key]
  select(index)
  tabs.value?.querySelector<HTMLElement>(`[data-model="${index}"]`)?.focus({ preventScroll: true })
}
function updateVisibility() {
  pageVisible.value = !document.hidden
  if (document.hidden) settle()
}
function updateReduced() { reduced.value = media?.matches ?? false; if (reduced.value) settle() }
watch(locale, async () => { await nextTick(); sizeShape(rendered.value, false); positionIndicator(false) })

onMounted(() => {
  media = matchMedia('(prefers-reduced-motion: reduce)')
  updateVisibility()
  updateReduced(); media.addEventListener('change', updateReduced)
  sizeShape('key', false)
  observer = new IntersectionObserver(entries => { visible.value = entries[0]?.isIntersecting ?? false; if (!visible.value) settle() })
  observer.observe(root.value!)
  resize = new ResizeObserver(() => { sizeShape(rendered.value, false); positionIndicator(false) })
  resize.observe(root.value!)
  document.addEventListener('visibilitychange', updateVisibility)
})
onBeforeUnmount(() => {
  disposed = true; revision++; clearRequest()
  for (const animation of animations) animation.cancel()
  animations.clear(); observer?.disconnect(); resize?.disconnect()
  media?.removeEventListener('change', updateReduced)
  document.removeEventListener('visibilitychange', updateVisibility)
})
</script>

<style scoped>
@font-face { font-family: 'Geist Demo'; src: url('../../assets/fonts/Geist.ttf') format('truetype'); font-style: normal; font-weight: 100 900; font-display: swap; }
.mg-demo { overflow-anchor: none; width: min(360px, 100%); min-width: 0; color: #202020; font-family: 'Geist Demo', 'PingFang SC', 'Microsoft YaHei', system-ui, sans-serif; -webkit-font-smoothing: antialiased; }
.mg-demo *, .mg-demo *::before, .mg-demo *::after { box-sizing: border-box; }
.mg-meta { display: flex; align-items: center; justify-content: space-between; height: 32px; padding: 0 4px; color: #62615d; font-size: 12px; letter-spacing: .04em; }
.mg-meta > span { display: inline-flex; align-items: center; gap: 7px; }
.mg-dot { width: 5px; height: 5px; border-radius: 50%; background: #777671; }
.mg-demo button { appearance: none; font: inherit; cursor: pointer; border: 0; -webkit-tap-highlight-color: transparent; }
.mg-reset { width: 44px; height: 44px; display: grid; place-items: center; background: transparent; color: #74736e; border-radius: 12px; }
.mg-demo :deep(svg) { width: 18px; height: 18px; flex: none; }
.mg-stage { position: relative; height: 352px; }
.mg-shape { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 360px; max-width: 100%; height: 258px; background: #191919; border-radius: 24px; overflow: clip; border: 1px solid rgb(0 0 0 / .08); box-shadow: 0 14px 30px -22px rgb(0 0 0 / .2); }
.mg-content { position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%); width: calc(100% - 40px); color: #202020; padding: 0; }
.mg-light-text { color: #fafaf9; }
.mg-topline { display: flex; align-items: center; gap: 8px; height: 24px; margin-bottom: 14px; font-size: 12px; font-weight: 500; }
.mg-example, .mg-step { margin-left: auto; font-size: 12px; letter-spacing: .02em; color: #66665f; }
.mg-example { border: 1px solid #535353; color: #bbbcb7; border-radius: 5px; padding: 2px 5px; }
.mg-eyebrow { color: #66665f; font-size: 12px; }
.mg-content h3 { margin: 0; font-size: 22px; letter-spacing: -.035em; font-weight: 550; line-height: 1.35; }
.mg-subtitle { margin: 6px 0 0; color: #bebebb; font-size: 12px; line-height: 1.6; }
.mg-subtitle span { color: #636363; padding: 0 5px; }
.mg-key { display: flex; justify-content: space-between; gap: 8px; align-items: center; margin: 16px 0; padding: 11px 0; border-top: 1px solid #393939; border-bottom: 1px solid #393939; font-size: 12px; color: #aaa; font-variant-numeric: tabular-nums; letter-spacing: .05em; }
.mg-key > span:last-child { font-size: 10px; letter-spacing: .12em; }
.mg-primary, .mg-secondary { min-height: 44px; display: flex; align-items: center; justify-content: center; gap: 10px; padding: 10px 14px; border-radius: 10px; font-size: 12px !important; font-weight: 550 !important; line-height: 1.4; transition: background-color 180ms ease, color 180ms ease; }
.mg-primary { width: 100%; background: #202020; color: #fff; }
.mg-light-text .mg-primary { background: #f9f9f7; color: #202020; }
.mg-tabs { margin-top: 14px; padding: 4px; position: relative; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); background: #efeeeb; border-radius: 13px; }
.mg-tabs button { position: relative; min-width: 0; min-height: 44px; padding: 0 3px; border-radius: 10px; background: transparent; font-size: 12px; color: #666; transition: color 180ms ease; }
.mg-tabs button[aria-checked=true] { color: #fff; }
.mg-indicator { position: absolute; top: 4px; bottom: 4px; left: 4px; border-radius: 10px; background: #202020; pointer-events: none; }
.mg-model-detail { margin: 14px 0 12px; }
.mg-small-label { color: #65655f; font-size: 12px; }
.mg-model-detail p { margin: 7px 0 0; font-size: 13px; line-height: 1.7; }
.mg-request-note { display: flex; gap: 6px; align-items: center; margin-bottom: 14px; color: #65655f; font-size: 12px; }
.mg-request-note :deep(svg) { width: 14px; height: 14px; }
.mg-loading-row, .mg-success-row { display: flex; gap: 14px; align-items: center; }
.mg-loading-row h3, .mg-success-row h3 { font-size: 16px; letter-spacing: -.02em; }
.mg-loading-row .mg-subtitle, .mg-success-row .mg-subtitle { font-size: 12px; }
 .mg-mark-slot { display: block; width: 28px; height: 28px; flex: none; }
.mg-shared-mark { position: absolute; z-index: 2; top: 20px; left: 20px; width: 28px; height: 28px; pointer-events: none; color: #f5f5f2; }
.mg-shared-mark :deep(svg) { display: block; width: 28px; height: 28px; overflow: visible; }
.mg-shared-mark circle { transform-origin: 12px 12px; }
.mg-shared-mark.is-loading circle { animation: mg-spin 1100ms linear infinite; }
.mg-demo[data-active=false] .mg-shared-mark circle { animation-play-state: paused; }
.mg-text-action { display: flex; gap: 8px; align-items: center; justify-content: center; min-height: 44px; width: 100%; background: transparent; color: #c5c5c0; border-radius: 8px; font-size: 12px !important; margin-top: 10px; }
.mg-answer { margin: 14px 0; font-size: 14px; line-height: 1.8; color: #53534f; }
.mg-receipt { display: flex; gap: 6px; align-items: center; color: #65655f; font-size: 12px; border-top: 1px solid #e9e8e3; padding-top: 12px; margin: 0 0 14px; }
.mg-receipt :deep(svg) { width: 14px; height: 14px; }
.mg-actions { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
.mg-secondary { background: #efeeeb; color: #40403d; }
.mg-disclaimer { margin: 2px 0 0; min-height: 24px; text-align: center; color: #65645e; font-size: 12px; line-height: 1.65; text-wrap: balance; }
.mg-demo[data-locale=en] .mg-stage { height: 376px; }
@media (max-width: 740px) { .mg-stage { height: 340px; } .mg-meta { height: 32px; } .mg-disclaimer { margin-top: 0; padding: 0 8px; } }
@media (max-width: 380px) { .mg-demo[data-locale=en] .mg-stage { height: 376px; } }
.mg-demo button:focus-visible, .mg-demo [tabindex='-1']:focus-visible { outline: 2px solid #858580; outline-offset: 3px; }
.mg-light-text button:focus-visible { outline-color: #fff; }
.mg-sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
@media (hover: hover) { .mg-primary:hover { background: #3b3b3b; } .mg-light-text .mg-primary:hover { background: #deded9; } .mg-secondary:hover { background: #e2e1dc; } .mg-reset:hover { background: #e9e7e1; } }
@media (max-width: 380px) { .mg-content h3 { font-size: 20px; } .mg-content .mg-loading-row h3, .mg-content .mg-success-row h3 { font-size: 15px; } .mg-tabs button { font-size: 11px; } .mg-key { letter-spacing: 0; } .mg-key > span:last-child { letter-spacing: .03em; } .mg-answer { font-size: 13px; } }
@media (prefers-reduced-motion: reduce) { .mg-demo *, .mg-demo *::before { animation: none !important; transition: none !important; } }
@keyframes mg-spin { to { transform: rotate(360deg); } }
</style>
