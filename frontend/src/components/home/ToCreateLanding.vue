<template>
  <div class="tc-home">
    <header class="tc-header">
      <nav class="tc-nav" aria-label="Main navigation">
        <router-link to="/" class="tc-wordmark" :aria-label="siteName">
          <img :src="siteLogo || '/logo.svg'" alt="" />
          <span>{{ siteName }}</span>
        </router-link>

        <div class="tc-nav-actions">
          <LocaleSwitcher />
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="tc-icon-button tc-model-link"
            :title="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="sm" />
            <span>{{ t('nav.modelPlaza') }}</span>
          </router-link>
          <button
            type="button"
            class="tc-icon-button"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="$emit('toggle-theme')"
          >
            <Icon v-if="isDark" name="sun" size="sm" />
            <Icon v-else name="moon" size="sm" />
          </button>
          <router-link v-if="isAuthenticated" :to="dashboardPath" class="tc-login-button">
            <span v-if="userInitial" class="tc-user-initial">{{ userInitial }}</span>
            {{ t('home.dashboard') }}
            <Icon name="arrowRight" size="xs" />
          </router-link>
          <button v-else type="button" class="tc-login-button" @click="authDialogOpen = true">
            {{ t('home.login') }}
            <Icon name="arrowRight" size="xs" />
          </button>
        </div>
      </nav>
    </header>

    <main>
      <section class="tc-hero">
        <div class="tc-hero-copy">
          <h1>{{ siteName }}</h1>
          <h2>{{ t('home.heroSubtitle') }}</h2>
          <p class="tc-hero-description">{{ t('home.heroDescription') }}</p>

          <div class="tc-hero-actions">
            <router-link v-if="isAuthenticated" :to="dashboardPath" class="tc-primary-action">
              {{ t('home.goToDashboard') }}
              <Icon name="arrowRight" size="sm" />
            </router-link>
            <button v-else type="button" class="tc-primary-action" @click="authDialogOpen = true">
              {{ t('home.login') }}
              <Icon name="arrowRight" size="sm" />
            </button>
          </div>
        </div>

        <div class="tc-hero-visual" :aria-label="t('home.redesign.gatewayPreview')">
          <article class="tc-map-card">
            <p>{{ t('home.redesign.diagramTitle') }}</p>
            <ol>
              <li>
                <span class="tc-provider-mark tc-provider-claude">C</span>
                Claude
              </li>
              <li>
                <span class="tc-provider-mark tc-provider-gpt">O</span>
                GPT
              </li>
              <li>
                <span class="tc-provider-mark tc-provider-gemini">G</span>
                Gemini
              </li>
            </ol>
          </article>
        </div>
      </section>
    </main>

    <footer class="tc-footer">
      <span class="tc-wordmark tc-footer-mark">
        <img :src="siteLogo || '/logo.svg'" alt="" />
        <span>{{ siteName }}</span>
      </span>
      <p>&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
    </footer>

    <EmailFirstAuthDialog
      v-model:open="authDialogOpen"
      :site-name="siteName"
      :site-logo="siteLogo"
      :dashboard-path="dashboardPath"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import EmailFirstAuthDialog from '@/components/auth/EmailFirstAuthDialog.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  siteName: string
  siteLogo: string
  siteSubtitle: string
  docUrl: string
  isAuthenticated: boolean
  dashboardPath: string
  userInitial: string
  isDark: boolean
  currentYear: number
  showModelPlazaEntry: boolean
}>()

defineEmits<{
  (event: 'toggle-theme'): void
}>()

const { t } = useI18n()
const authDialogOpen = ref(false)
</script>

<style scoped>
.tc-home {
  --tc-paper: #f6f4ef;
  --tc-surface: #fffefb;
  --tc-ink: #262823;
  --tc-muted: #6f726c;
  --tc-subtle: #999b94;
  --tc-line: rgba(42, 47, 40, 0.12);
  --tc-brand: #aa7149;
  --tc-brand-deep: #895634;
  min-height: 100vh;
  overflow-x: hidden;
  color: var(--tc-ink);
  background: var(--tc-paper);
  font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

:global(html.dark .tc-home) {
  --tc-paper: #171916;
  --tc-surface: #232620;
  --tc-ink: #f1eee7;
  --tc-muted: #b6b7b0;
  --tc-subtle: #858981;
  --tc-line: rgba(240, 238, 230, 0.12);
  --tc-brand: #d09a71;
  --tc-brand-deep: #e3b48f;
}

.tc-home *,
.tc-home *::before,
.tc-home *::after {
  box-sizing: border-box;
}

.tc-header {
  position: sticky;
  z-index: 30;
  top: 0;
  border-bottom: 1px solid var(--tc-line);
  background: color-mix(in srgb, var(--tc-paper) 92%, transparent);
  backdrop-filter: blur(20px);
}

.tc-nav {
  width: min(1180px, calc(100% - 48px));
  min-height: 66px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.tc-wordmark {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: var(--tc-ink);
  font-family: Georgia, "Times New Roman", serif;
  font-size: 19px;
  font-weight: 600;
}

.tc-wordmark img {
  width: 29px;
  height: 29px;
  flex: none;
  border: 1px solid var(--tc-line);
  border-radius: 7px;
  object-fit: contain;
  background: var(--tc-surface);
}

.tc-nav-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.tc-icon-button {
  width: 34px;
  height: 34px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  border-radius: 7px;
  color: var(--tc-muted);
  background: transparent;
}

.tc-model-link {
  width: auto;
  gap: 6px;
  padding: 0 9px;
  font-size: 12px;
}

.tc-login-button,
.tc-primary-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  font-weight: 650;
}

.tc-login-button {
  min-height: 36px;
  gap: 7px;
  padding: 0 14px;
  border: 0;
  color: #fff;
  background: var(--tc-ink);
  cursor: pointer;
  font-size: 12px;
}

:global(html.dark .tc-login-button) {
  color: #1d211d;
  background: #f0eee8;
}

.tc-user-initial {
  width: 20px;
  height: 20px;
  display: grid;
  border-radius: 50%;
  color: #fff;
  background: #237a70;
  font-size: 10px;
  place-items: center;
}

.tc-hero {
  width: min(1180px, calc(100% - 48px));
  min-height: calc(100vh - 160px);
  margin: 0 auto;
  display: grid;
  grid-template-columns: 0.94fr 1.06fr;
  align-items: center;
  gap: 68px;
  padding: 76px 0 60px;
}

.tc-hero h1 {
  margin: 0 0 8px;
  color: var(--tc-ink);
  font-family: Georgia, "Times New Roman", serif;
  font-size: 70px;
  font-weight: 500;
  line-height: 1;
  overflow-wrap: anywhere;
}

.tc-hero h2 {
  max-width: 520px;
  margin: 0;
  color: var(--tc-brand);
  font-family: Georgia, "Times New Roman", serif;
  font-size: 36px;
  font-style: italic;
  font-weight: 400;
  line-height: 1.2;
  overflow-wrap: anywhere;
}

.tc-hero-description {
  max-width: 420px;
  margin: 20px 0 0;
  color: var(--tc-muted);
  font-size: 16px;
  line-height: 1.7;
}

.tc-hero-actions {
  display: flex;
  align-items: center;
  gap: 11px;
  margin-top: 30px;
}

.tc-primary-action {
  min-height: 48px;
  min-width: 132px;
  gap: 12px;
  padding: 0 20px;
  border: 0;
  color: #fff;
  background: var(--tc-brand-deep);
  box-shadow: 0 12px 24px color-mix(in srgb, var(--tc-brand-deep) 20%, transparent);
  cursor: pointer;
  font-size: 13px;
}

.tc-hero-visual {
  display: flex;
  align-items: center;
  justify-content: center;
}

.tc-map-card {
  width: min(360px, 100%);
  padding: 28px 28px 22px;
  border: 1px solid var(--tc-line);
  border-radius: 10px;
  background: color-mix(in srgb, var(--tc-surface) 94%, transparent);
  box-shadow: 0 22px 48px rgba(58, 47, 37, 0.1);
}

.tc-map-card > p {
  margin: 0 0 18px;
  color: var(--tc-subtle);
  font-size: 12px;
}

.tc-map-card ol {
  margin: 0;
  padding: 0;
  list-style: none;
}

.tc-map-card li {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 52px;
  padding: 8px 0;
  border-top: 1px solid var(--tc-line);
  color: var(--tc-ink);
  font-size: 16px;
}

.tc-provider-mark {
  width: 28px;
  height: 28px;
  display: grid;
  flex: none;
  border-radius: 7px;
  color: #fff;
  font-family: Georgia, "Times New Roman", serif;
  font-size: 12px;
  font-weight: 700;
  place-items: center;
}

.tc-provider-claude { background: #c77642; }
.tc-provider-gpt { background: #288066; }
.tc-provider-gemini { background: #4578c8; }

.tc-footer {
  width: min(1180px, calc(100% - 48px));
  min-height: 82px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  border-top: 1px solid var(--tc-line);
  color: var(--tc-subtle);
  font-size: 11px;
}

.tc-footer-mark {
  font-size: 15px;
}

.tc-footer-mark img {
  width: 24px;
  height: 24px;
}

@media (hover: hover) {
  .tc-icon-button:hover {
    border-color: var(--tc-line);
    color: var(--tc-ink);
    background: color-mix(in srgb, var(--tc-surface) 72%, transparent);
  }

  .tc-login-button:hover,
  .tc-primary-action:hover {
    box-shadow: 0 14px 28px color-mix(in srgb, var(--tc-brand-deep) 24%, transparent);
    transform: translateY(-1px);
  }
}

.tc-icon-button:active,
.tc-login-button:active,
.tc-primary-action:active {
  transform: scale(0.97);
}

.tc-icon-button:focus-visible,
.tc-login-button:focus-visible,
.tc-primary-action:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--tc-brand) 65%, transparent);
  outline-offset: 3px;
}

@media (max-width: 980px) {
  .tc-hero {
    grid-template-columns: 1fr;
    gap: 36px;
    min-height: auto;
    padding-top: 56px;
  }

  .tc-hero-copy {
    text-align: center;
  }

  .tc-hero h1 {
    font-size: 48px;
  }

  .tc-hero h2,
  .tc-hero-description {
    margin-left: auto;
    margin-right: auto;
  }

  .tc-hero-actions {
    justify-content: center;
  }
}

@media (max-width: 740px) {
  .tc-nav,
  .tc-hero,
  .tc-footer {
    width: min(100% - 32px, 560px);
  }

  .tc-wordmark > span:last-child,
  .tc-model-link span {
    display: none;
  }

  .tc-model-link {
    width: 34px;
    padding: 0;
  }

  .tc-hero {
    padding: 48px 0 40px;
  }

  .tc-hero h1 {
    font-size: 34px;
    line-height: 1.1;
  }

  .tc-hero h2 {
    font-size: 22px;
  }

  .tc-hero-description {
    font-size: 14px;
  }

  .tc-footer {
    flex-wrap: wrap;
    padding: 20px 0;
  }
}

@media (max-width: 460px) {
  .tc-nav-actions {
    gap: 4px;
  }

  .tc-nav-actions :deep(button:first-child) {
    max-width: 46px;
  }

  .tc-login-button {
    padding: 0 11px;
  }

  .tc-login-button svg {
    display: none;
  }

  .tc-primary-action {
    width: 100%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .tc-home *,
  .tc-home *::before,
  .tc-home *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
