<template>
  <div class="tc-auth-shell">
    <router-link to="/" class="tc-auth-home" :aria-label="siteName">
      <Icon name="arrowLeft" size="sm" />
      <span>{{ t('home.redesign.backHome') }}</span>
    </router-link>

    <div class="tc-auth-frame">
      <section class="tc-auth-intro">
        <div v-if="settingsLoaded" class="tc-auth-brand">
          <img :src="siteLogo || '/logo.svg'" alt="" />
          <span>{{ siteName }}</span>
        </div>

        <div class="tc-auth-copy">
          <p>{{ t('home.redesign.authEyebrow') }}</p>
          <h1>{{ t('home.redesign.authHeadline') }}</h1>
          <span>{{ t('home.redesign.authDescription') }}</span>
        </div>

        <div class="tc-auth-preview" aria-hidden="true">
          <p>{{ t('home.redesign.diagramTitle') }}</p>
          <ul>
            <li><span class="tc-auth-provider tc-auth-claude">C</span>Claude</li>
            <li><span class="tc-auth-provider tc-auth-gpt">O</span>GPT</li>
            <li><span class="tc-auth-provider tc-auth-gemini">G</span>Gemini</li>
          </ul>
        </div>
      </section>

      <section class="tc-auth-content">
        <div v-if="settingsLoaded" class="tc-auth-mobile-brand">
          <img :src="siteLogo || '/logo.svg'" alt="" />
          <span>{{ siteName }}</span>
        </div>

        <div class="tc-auth-form">
          <slot />
        </div>

        <div class="tc-auth-footer">
          <slot name="footer" />
        </div>

        <p class="tc-auth-copyright">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)
const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.tc-auth-shell {
  --auth-paper: #f3f1eb;
  --auth-surface: #fffefb;
  --auth-ink: #272923;
  --auth-muted: #72756e;
  --auth-subtle: #9b9d96;
  --auth-line: rgba(42, 47, 40, 0.12);
  --auth-brand: #9a623d;
  --auth-teal: #287d72;
  position: relative;
  min-height: 100vh;
  display: grid;
  padding: 42px 24px;
  overflow: hidden;
  color: var(--auth-ink);
  background: var(--auth-paper);
  place-items: center;
}

:global(html.dark .tc-auth-shell) {
  --auth-paper: #171916;
  --auth-surface: #232620;
  --auth-ink: #f1eee7;
  --auth-muted: #b7b9b1;
  --auth-subtle: #878b83;
  --auth-line: rgba(240, 238, 230, 0.12);
  --auth-brand: #d09a71;
  --auth-teal: #76b9ad;
}

.tc-auth-shell *,
.tc-auth-shell *::before,
.tc-auth-shell *::after {
  box-sizing: border-box;
  letter-spacing: 0;
}

.tc-auth-home {
  position: absolute;
  z-index: 3;
  top: 24px;
  left: 28px;
  display: inline-flex;
  align-items: center;
  gap: 7px;
  color: var(--auth-muted);
  font-size: 12px;
  font-weight: 600;
  transition: color 160ms ease, transform 160ms ease;
}

.tc-auth-frame {
  position: relative;
  z-index: 1;
  width: min(1040px, 100%);
  min-height: 620px;
  display: grid;
  grid-template-columns: 0.9fr 1.1fr;
  overflow: hidden;
  border: 1px solid var(--auth-line);
  border-radius: 8px;
  background: var(--auth-surface);
  box-shadow: 0 34px 86px rgba(46, 38, 31, 0.15), 0 7px 24px rgba(46, 38, 31, 0.07);
}

:global(html.dark .tc-auth-frame) {
  box-shadow: 0 34px 86px rgba(0, 0, 0, 0.38), 0 7px 24px rgba(0, 0, 0, 0.22);
}

.tc-auth-intro {
  position: relative;
  min-width: 0;
  display: flex;
  flex-direction: column;
  padding: 42px 46px;
  overflow: hidden;
  color: #ece9e2;
  background: #292b27;
}

.tc-auth-brand,
.tc-auth-mobile-brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-family: Georgia, "Times New Roman", serif;
  font-size: 19px;
  font-weight: 600;
}

.tc-auth-brand img,
.tc-auth-mobile-brand img {
  width: 32px;
  height: 32px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 7px;
  object-fit: contain;
  background: #fffefb;
}

.tc-auth-copy {
  position: relative;
  z-index: 1;
  margin-top: 78px;
}

.tc-auth-copy > p {
  margin: 0 0 14px;
  color: #d3a077;
  font-size: 9px;
  font-weight: 800;
  text-transform: uppercase;
}

.tc-auth-copy h1 {
  margin: 0;
  font-family: Georgia, "Times New Roman", serif;
  font-size: 43px;
  font-weight: 400;
  line-height: 1.08;
}

.tc-auth-copy > span {
  display: block;
  max-width: 330px;
  margin-top: 16px;
  color: #aeb0a9;
  font-size: 13px;
  line-height: 1.7;
}

.tc-auth-preview {
  position: relative;
  z-index: 1;
  margin-top: 34px;
  padding: 18px 16px 10px;
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.09);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.035);
}

.tc-auth-preview > p {
  margin: 0 0 10px;
  color: #a6a8a1;
  font-size: 12px;
}

.tc-auth-preview ul {
  margin: 0;
  padding: 0;
  list-style: none;
}

.tc-auth-preview li {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 42px;
  border-top: 1px solid rgba(255, 255, 255, 0.07);
  color: #e9e6df;
  font-size: 14px;
}

.tc-auth-provider {
  width: 28px;
  height: 28px;
  display: grid;
  border-radius: 7px;
  color: #fff;
  font-family: Georgia, "Times New Roman", serif;
  font-size: 11px;
  font-weight: 700;
  place-items: center;
}

.tc-auth-claude { background: #bd7247; }
.tc-auth-gpt { background: #2f7c68; }
.tc-auth-gemini { background: #4578c8; }

.tc-auth-content {
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 48px 64px 32px;
  background: var(--auth-surface);
}

.tc-auth-mobile-brand {
  display: none;
  margin-bottom: 30px;
  color: var(--auth-ink);
}

.tc-auth-mobile-brand img {
  border-color: var(--auth-line);
}

.tc-auth-form {
  width: 100%;
  max-width: 390px;
  margin: auto;
}

.tc-auth-form :deep(h2) {
  color: var(--auth-ink) !important;
  font-family: Georgia, "Times New Roman", serif;
  font-size: 31px !important;
  font-weight: 500 !important;
  line-height: 1.12;
}

.tc-auth-form :deep(.text-gray-500),
.tc-auth-form :deep(.dark\:text-dark-400) {
  color: var(--auth-muted) !important;
}

.tc-auth-form :deep(.input-label) {
  color: var(--auth-ink);
  font-size: 11px;
  font-weight: 700;
}

.tc-auth-form :deep(.input) {
  min-height: 47px;
  border-color: var(--auth-line);
  border-radius: 8px;
  color: var(--auth-ink);
  background: color-mix(in srgb, var(--auth-paper) 52%, var(--auth-surface));
  box-shadow: inset 0 1px 2px rgba(49, 41, 34, 0.025);
}

.tc-auth-form :deep(.input:focus) {
  border-color: color-mix(in srgb, var(--auth-brand) 62%, transparent);
  background: var(--auth-surface);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--auth-brand) 11%, transparent);
}

.tc-auth-form :deep(.btn) {
  min-height: 47px;
  border-radius: 8px;
}

.tc-auth-form :deep(.btn-primary) {
  color: #fff;
  background: var(--auth-brand);
  box-shadow: 0 10px 20px color-mix(in srgb, var(--auth-brand) 19%, transparent);
}

.tc-auth-form :deep(.btn-primary:hover:not(:disabled)) {
  background: color-mix(in srgb, var(--auth-brand) 88%, #2c241e);
  box-shadow: 0 13px 25px color-mix(in srgb, var(--auth-brand) 24%, transparent);
  transform: translateY(-1px);
}

.tc-auth-form :deep(.btn-secondary) {
  border-color: var(--auth-line);
  color: var(--auth-ink);
  background: var(--auth-surface);
}

.tc-auth-form :deep(.text-primary-600),
.tc-auth-form :deep(.text-primary-500),
.tc-auth-form :deep(.dark\:text-primary-400) {
  color: var(--auth-brand) !important;
}

.tc-auth-form :deep(.focus\:ring-primary-500\/50:focus),
.tc-auth-form :deep(.focus\:ring-primary-500\/30:focus) {
  --tw-ring-color: color-mix(in srgb, var(--auth-brand) 32%, transparent) !important;
}

.tc-auth-footer {
  min-height: 30px;
  margin-top: 22px;
  color: var(--auth-muted);
  text-align: center;
  font-size: 12px;
}

.tc-auth-footer :deep(a) {
  color: var(--auth-brand) !important;
}

.tc-auth-copyright {
  margin: 18px 0 0;
  color: var(--auth-subtle);
  font-size: 8px;
  text-align: center;
}

@media (hover: hover) {
  .tc-auth-home:hover {
    color: var(--auth-ink);
    transform: translateX(-2px);
  }
}

.tc-auth-home:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--auth-brand) 65%, transparent);
  outline-offset: 3px;
}

@keyframes tc-auth-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.55; transform: scale(0.8); }
}

@keyframes tc-auth-signal {
  from { left: -10px; opacity: 0; }
  12% { opacity: 1; }
  88% { opacity: 1; }
  to { left: 100%; opacity: 0; }
}

@media (max-width: 820px) {
  .tc-auth-shell {
    align-items: start;
    padding: 72px 16px 24px;
  }

  .tc-auth-home {
    top: 22px;
    left: 20px;
  }

  .tc-auth-frame {
    width: min(520px, 100%);
    min-height: 0;
    grid-template-columns: 1fr;
  }

  .tc-auth-intro {
    display: none;
  }

  .tc-auth-content {
    padding: 38px 34px 28px;
  }

  .tc-auth-mobile-brand {
    display: inline-flex;
  }
}

@media (max-width: 460px) {
  .tc-auth-shell {
    align-items: end;
    padding: 62px 8px 8px;
  }

  .tc-auth-home span {
    display: none;
  }

  .tc-auth-frame {
    width: 100%;
    border-radius: 8px;
  }

  .tc-auth-content {
    padding: 30px 24px 22px;
  }

  .tc-auth-mobile-brand {
    margin-bottom: 26px;
  }

  .tc-auth-form :deep(h2) {
    font-size: 29px !important;
  }
}

@media (prefers-reduced-motion: reduce) {
  .tc-auth-shell *,
  .tc-auth-shell *::before,
  .tc-auth-shell *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
