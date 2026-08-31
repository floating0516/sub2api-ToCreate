<template>
  <div class="tc-home">
    <header class="tc-header">
      <nav class="tc-nav" aria-label="Main navigation">
        <router-link to="/" class="tc-wordmark" :aria-label="siteName">
          <img :src="siteLogo || '/logo.svg'" alt="" />
          <span>{{ siteName }}</span>
        </router-link>

        <div class="tc-nav-links">
          <a href="#capabilities">{{ t('home.solutions.title') }}</a>
          <a href="#models">{{ t('home.providers.title') }}</a>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">
            {{ t('home.docs') }}
          </a>
        </div>

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
          <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="tc-login-button">
            <span v-if="isAuthenticated" class="tc-user-initial">{{ userInitial }}</span>
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
            <Icon name="arrowRight" size="xs" />
          </router-link>
        </div>
      </nav>
    </header>

    <main>
      <section class="tc-hero">
        <div class="tc-hero-copy">
          <div class="tc-live-badge">
            <span class="tc-live-dot"><i /></span>
            {{ t('home.redesign.liveGateway') }}
            <span class="tc-badge-divider" />
            {{ t('home.redesign.fourProviders') }}
          </div>

          <h1>{{ siteName }}</h1>
          <h2>{{ t('home.heroSubtitle') }}</h2>
          <p class="tc-hero-description">{{ t('home.heroDescription') }}</p>

          <div class="tc-hero-actions">
            <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="tc-primary-action">
              {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
              <Icon name="arrowRight" size="sm" />
            </router-link>
            <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="tc-secondary-action">
              <Icon name="book" size="sm" />
              {{ t('home.viewDocs') }}
            </a>
            <a v-else href="#capabilities" class="tc-secondary-action">
              {{ t('home.solutions.title') }}
            </a>
          </div>

          <div class="tc-proof-row">
            <span><Icon name="check" size="xs" /> {{ t('home.tags.subscriptionToApi') }}</span>
            <span><Icon name="check" size="xs" /> {{ t('home.tags.stickySession') }}</span>
            <span><Icon name="check" size="xs" /> {{ t('home.tags.realtimeBilling') }}</span>
          </div>
        </div>

        <div class="tc-hero-visual" :aria-label="t('home.redesign.gatewayPreview')">
          <div class="tc-service-chip tc-service-claude">
            <span class="tc-provider-mark tc-provider-claude">C</span>
            <span><small>{{ t('home.redesign.connected') }}</small>Claude</span>
            <i />
          </div>
          <div class="tc-service-chip tc-service-gpt">
            <span class="tc-provider-mark tc-provider-gpt">G</span>
            <span><small>{{ t('home.redesign.routing') }}</small>GPT</span>
            <i />
          </div>

          <article class="tc-gateway-card">
            <div class="tc-gateway-topline">
              <div class="tc-window-dots" aria-hidden="true"><i /><i /><i /></div>
              <span><Icon name="shield" size="xs" /> {{ t('home.redesign.secured') }}</span>
            </div>

            <div class="tc-gateway-heading">
              <div>
                <p>{{ t('home.redesign.gatewayStatus') }}</p>
                <strong>{{ t('home.redesign.operational') }}</strong>
              </div>
              <span class="tc-uptime"><i /> 99.98%</span>
            </div>

            <div class="tc-route-chart" aria-hidden="true">
              <svg viewBox="0 0 460 112" preserveAspectRatio="none">
                <path class="tc-chart-grid" d="M0 24H460M0 56H460M0 88H460" />
                <path class="tc-chart-line" d="M0 88 C42 83 56 89 91 72 C126 55 145 70 181 55 C216 41 234 51 267 34 C301 17 326 34 358 20 C393 5 417 19 460 7" />
                <circle class="tc-chart-halo" cx="460" cy="7" r="8" />
                <circle class="tc-chart-point" cx="460" cy="7" r="3.5" />
              </svg>
              <div><span>00:00</span><span>06:00</span><span>12:00</span><span>18:00</span><span>NOW</span></div>
            </div>

            <div class="tc-request-list">
              <div class="tc-request-header">
                <span><Icon name="bolt" size="xs" /> {{ t('home.redesign.liveRequests') }}</span>
                <small>{{ t('home.redesign.justNow') }}</small>
              </div>
              <div class="tc-request-row">
                <span class="tc-request-icon"><Icon name="sparkles" size="sm" /></span>
                <span class="tc-request-main">claude-sonnet-4<small>8,420 tokens</small></span>
                <strong>200 OK</strong>
              </div>
              <div class="tc-request-row tc-request-delayed">
                <span class="tc-request-icon"><Icon name="cpu" size="sm" /></span>
                <span class="tc-request-main">gpt-5.2-codex<small>3,106 tokens</small></span>
                <strong>200 OK</strong>
              </div>
            </div>
          </article>

          <div class="tc-route-pulse">
            <span><Icon name="swap" size="sm" /></span>
            <span><small>{{ t('home.redesign.smartRoute') }}</small><strong>312 ms</strong></span>
          </div>
        </div>
      </section>

      <section id="models" class="tc-model-strip" :aria-label="t('home.providers.title')">
        <div><span class="tc-model-symbol tc-symbol-claude">C</span><strong>Claude</strong><small>{{ t('home.providers.supported') }}</small></div>
        <i />
        <div><span class="tc-model-symbol tc-symbol-gpt">G</span><strong>GPT</strong><small>{{ t('home.providers.supported') }}</small></div>
        <i />
        <div><span class="tc-model-symbol tc-symbol-gemini">G</span><strong>Gemini</strong><small>{{ t('home.providers.supported') }}</small></div>
        <i />
        <div><span class="tc-model-symbol tc-symbol-antigravity">A</span><strong>Antigravity</strong><small>{{ t('home.providers.supported') }}</small></div>
      </section>

      <section id="capabilities" class="tc-capabilities">
        <div class="tc-section-heading">
          <p>{{ t('home.redesign.quietInfrastructure') }}</p>
          <h2>{{ t('home.solutions.title') }}</h2>
          <span>{{ t('home.solutions.subtitle') }}</span>
        </div>

        <div class="tc-feature-grid">
          <article>
            <div class="tc-feature-top"><span><Icon name="key" size="md" /></span><small>01</small></div>
            <h3>{{ t('home.features.unifiedGateway') }}</h3>
            <p>{{ t('home.features.unifiedGatewayDesc') }}</p>
          </article>
          <article>
            <div class="tc-feature-top"><span><Icon name="swap" size="md" /></span><small>02</small></div>
            <h3>{{ t('home.features.multiAccount') }}</h3>
            <p>{{ t('home.features.multiAccountDesc') }}</p>
          </article>
          <article>
            <div class="tc-feature-top"><span><Icon name="chart" size="md" /></span><small>03</small></div>
            <h3>{{ t('home.features.balanceQuota') }}</h3>
            <p>{{ t('home.features.balanceQuotaDesc') }}</p>
          </article>
        </div>
      </section>

      <section class="tc-developer-band">
        <div class="tc-developer-copy">
          <p>{{ t('home.redesign.builtForWorkflow') }}</p>
          <h2>{{ t('home.redesign.oneEndpoint') }}</h2>
          <span>{{ siteSubtitle }}</span>
        </div>
        <div class="tc-terminal-card terminal-container">
          <div class="tc-terminal-bar"><span /><span /><span /><small>terminal</small></div>
          <code><i>$</i> curl {{ siteName.toLowerCase() }}.api/v1/messages</code>
          <p><Icon name="check" size="xs" /> {{ t('home.redesign.routeReady') }}</p>
          <p><Icon name="check" size="xs" /> {{ t('home.redesign.usageVisible') }}</p>
          <strong><i /> {{ t('home.redesign.readyForRequests') }}</strong>
        </div>
      </section>
    </main>

    <footer class="tc-footer">
      <span class="tc-wordmark tc-footer-mark">
        <img :src="siteLogo || '/logo.svg'" alt="" />
        <span>{{ siteName }}</span>
      </span>
      <p>&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
      <a :href="githubUrl" target="_blank" rel="noopener noreferrer">GitHub</a>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
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
  githubUrl: string
  showModelPlazaEntry: boolean
}>()

defineEmits<{
  (event: 'toggle-theme'): void
}>()

const { t } = useI18n()
</script>

<style scoped>
.tc-home {
  --tc-paper: #f6f4ef;
  --tc-paper-strong: #eeeae1;
  --tc-surface: #fffefb;
  --tc-ink: #262823;
  --tc-muted: #6f726c;
  --tc-subtle: #999b94;
  --tc-line: rgba(42, 47, 40, 0.12);
  --tc-brand: #aa7149;
  --tc-brand-deep: #895634;
  --tc-teal: #237a70;
  --tc-teal-soft: #e2efeb;
  min-height: 100vh;
  overflow: hidden;
  color: var(--tc-ink);
  background: var(--tc-paper);
  font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

:global(html.dark .tc-home) {
  --tc-paper: #171916;
  --tc-paper-strong: #20231f;
  --tc-surface: #232620;
  --tc-ink: #f1eee7;
  --tc-muted: #b6b7b0;
  --tc-subtle: #858981;
  --tc-line: rgba(240, 238, 230, 0.12);
  --tc-brand: #d09a71;
  --tc-brand-deep: #e3b48f;
  --tc-teal: #73b7aa;
  --tc-teal-soft: #263d38;
}

.tc-home *,
.tc-home *::before,
.tc-home *::after {
  box-sizing: border-box;
  letter-spacing: 0;
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
  height: 66px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
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

.tc-nav-links {
  display: flex;
  align-items: center;
  gap: 30px;
  margin-left: auto;
  color: var(--tc-muted);
  font-size: 13px;
  font-weight: 520;
}

.tc-nav-links a,
.tc-footer a {
  transition: color 160ms ease;
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
  transition: color 160ms ease, border-color 160ms ease, background 160ms ease, transform 150ms ease;
}

.tc-model-link {
  width: auto;
  gap: 6px;
  padding: 0 9px;
  font-size: 12px;
}

.tc-login-button,
.tc-primary-action,
.tc-secondary-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  font-weight: 650;
  transition: color 160ms ease, border-color 160ms ease, background 160ms ease, box-shadow 180ms ease, transform 150ms ease;
}

.tc-login-button {
  min-height: 36px;
  gap: 7px;
  padding: 0 14px;
  color: #fff;
  background: var(--tc-ink);
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
  background: var(--tc-teal);
  font-size: 10px;
  place-items: center;
}

.tc-hero {
  position: relative;
  width: min(1180px, calc(100% - 48px));
  min-height: 655px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 0.94fr 1.06fr;
  align-items: center;
  gap: 68px;
  padding: 76px 0 60px;
}

.tc-hero-copy,
.tc-hero-visual {
  position: relative;
  z-index: 1;
}

.tc-live-badge {
  width: max-content;
  display: flex;
  align-items: center;
  gap: 9px;
  margin-bottom: 24px;
  padding: 7px 10px;
  border: 1px solid color-mix(in srgb, var(--tc-teal) 24%, transparent);
  border-radius: 999px;
  color: var(--tc-muted);
  background: color-mix(in srgb, var(--tc-surface) 72%, transparent);
  font-size: 9px;
  font-weight: 760;
  text-transform: uppercase;
}

.tc-live-dot {
  width: 9px;
  height: 9px;
  display: grid;
  border-radius: 50%;
  background: color-mix(in srgb, var(--tc-teal) 18%, transparent);
  place-items: center;
}

.tc-live-dot i {
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--tc-teal);
  animation: tc-status-pulse 2s ease-in-out infinite;
}

.tc-badge-divider {
  width: 1px;
  height: 11px;
  background: var(--tc-line);
}

.tc-hero h1 {
  margin: 0 0 8px;
  color: var(--tc-ink);
  font-family: Georgia, "Times New Roman", serif;
  font-size: 70px;
  font-weight: 500;
  line-height: 1;
}

.tc-hero h2 {
  max-width: 560px;
  margin: 0;
  color: var(--tc-brand);
  font-family: Georgia, "Times New Roman", serif;
  font-size: 47px;
  font-style: italic;
  font-weight: 400;
  line-height: 1.05;
}

.tc-hero-description {
  max-width: 510px;
  margin: 24px 0 0;
  color: var(--tc-muted);
  font-size: 16px;
  line-height: 1.75;
}

.tc-hero-actions {
  display: flex;
  align-items: center;
  gap: 11px;
  margin-top: 30px;
}

.tc-primary-action,
.tc-secondary-action {
  min-height: 48px;
  gap: 12px;
  padding: 0 20px;
  font-size: 13px;
}

.tc-primary-action {
  min-width: 154px;
  color: #fff;
  background: var(--tc-brand-deep);
  box-shadow: 0 12px 24px color-mix(in srgb, var(--tc-brand-deep) 20%, transparent);
}

.tc-secondary-action {
  border: 1px solid var(--tc-line);
  color: var(--tc-ink);
  background: color-mix(in srgb, var(--tc-surface) 76%, transparent);
}

.tc-proof-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 19px;
  color: var(--tc-subtle);
  font-size: 10px;
}

.tc-proof-row span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.tc-proof-row svg {
  color: var(--tc-teal);
}

.tc-hero-visual {
  min-height: 490px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tc-gateway-card {
  position: relative;
  z-index: 3;
  width: min(490px, 88%);
  padding: 18px 20px 20px;
  border: 1px solid var(--tc-line);
  border-radius: 8px;
  background: color-mix(in srgb, var(--tc-surface) 94%, transparent);
  box-shadow: 0 30px 65px rgba(58, 47, 37, 0.13), 0 5px 16px rgba(58, 47, 37, 0.06);
  backdrop-filter: blur(18px);
  transform: rotateY(-2deg) rotateX(1deg);
  transition: transform 380ms ease, box-shadow 380ms ease;
}

:global(html.dark .tc-gateway-card) {
  box-shadow: 0 30px 65px rgba(0, 0, 0, 0.34), 0 5px 16px rgba(0, 0, 0, 0.2);
}

.tc-gateway-topline,
.tc-gateway-heading,
.tc-request-header,
.tc-request-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.tc-gateway-topline {
  padding-bottom: 15px;
  border-bottom: 1px solid var(--tc-line);
  color: var(--tc-subtle);
  font-size: 8px;
  font-weight: 750;
  text-transform: uppercase;
}

.tc-gateway-topline > span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.tc-gateway-topline svg {
  color: var(--tc-teal);
}

.tc-window-dots {
  display: flex;
  gap: 5px;
}

.tc-window-dots i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--tc-subtle) 55%, transparent);
}

.tc-gateway-heading {
  padding: 23px 2px 8px;
  align-items: flex-start;
}

.tc-gateway-heading p {
  margin: 0 0 5px;
  color: var(--tc-subtle);
  font-size: 10px;
}

.tc-gateway-heading strong {
  color: var(--tc-ink);
  font-family: Georgia, "Times New Roman", serif;
  font-size: 31px;
  font-weight: 500;
}

.tc-uptime {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 13px;
  padding: 5px 7px;
  border-radius: 6px;
  color: var(--tc-teal);
  background: var(--tc-teal-soft);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 9px;
  font-weight: 750;
}

.tc-uptime i {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--tc-teal);
}

.tc-route-chart {
  padding: 0 3px 13px;
}

.tc-route-chart svg {
  width: 100%;
  height: 112px;
  overflow: visible;
}

.tc-chart-grid {
  fill: none;
  stroke: var(--tc-line);
  stroke-width: 1;
  stroke-dasharray: 2 5;
}

.tc-chart-line {
  fill: none;
  stroke: var(--tc-brand);
  stroke-width: 2;
  stroke-linecap: round;
  stroke-dasharray: 700;
  animation: tc-draw-line 1.8s ease both 300ms;
}

.tc-chart-halo {
  fill: color-mix(in srgb, var(--tc-brand) 18%, transparent);
  animation: tc-point-pulse 2s ease-out infinite;
  transform-origin: 460px 7px;
}

.tc-chart-point {
  fill: var(--tc-surface);
  stroke: var(--tc-brand);
  stroke-width: 2;
}

.tc-route-chart > div {
  display: flex;
  justify-content: space-between;
  color: var(--tc-subtle);
  font-size: 7px;
  font-weight: 700;
}

.tc-request-list {
  overflow: hidden;
  border: 1px solid var(--tc-line);
  border-radius: 8px;
  background: color-mix(in srgb, var(--tc-paper) 55%, var(--tc-surface));
}

.tc-request-header {
  min-height: 32px;
  padding: 0 11px;
  border-bottom: 1px solid var(--tc-line);
  color: var(--tc-muted);
  font-size: 8px;
  font-weight: 750;
  text-transform: uppercase;
}

.tc-request-header span {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.tc-request-header svg {
  color: var(--tc-teal);
}

.tc-request-header small {
  color: var(--tc-subtle);
  font-size: 7px;
}

.tc-request-row {
  min-height: 52px;
  gap: 9px;
  justify-content: flex-start;
  padding: 8px 11px;
  border-bottom: 1px solid var(--tc-line);
  animation: tc-request-in 500ms ease both 900ms;
}

.tc-request-row:last-child {
  border-bottom: 0;
}

.tc-request-delayed {
  animation-delay: 1.2s;
}

.tc-request-icon {
  width: 29px;
  height: 29px;
  display: grid;
  flex: none;
  border: 1px solid var(--tc-line);
  border-radius: 7px;
  color: var(--tc-brand);
  background: var(--tc-surface);
  place-items: center;
}

.tc-request-main {
  min-width: 0;
  flex: 1;
  color: var(--tc-ink);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 9px;
}

.tc-request-main small {
  display: block;
  margin-top: 3px;
  color: var(--tc-subtle);
  font-family: system-ui, sans-serif;
  font-size: 7px;
}

.tc-request-row > strong {
  color: var(--tc-teal);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 8px;
}

.tc-service-chip,
.tc-route-pulse {
  position: absolute;
  z-index: 5;
  display: flex;
  align-items: center;
  border: 1px solid var(--tc-line);
  border-radius: 8px;
  background: color-mix(in srgb, var(--tc-surface) 94%, transparent);
  box-shadow: 0 13px 30px rgba(55, 44, 35, 0.1);
  backdrop-filter: blur(16px);
}

.tc-service-chip {
  width: 145px;
  gap: 9px;
  padding: 9px 10px;
  animation: tc-float 5s ease-in-out infinite;
}

.tc-service-chip > span:nth-child(2) {
  flex: 1;
  color: var(--tc-ink);
  font-size: 10px;
  font-weight: 650;
}

.tc-service-chip small,
.tc-route-pulse small {
  display: block;
  margin-bottom: 2px;
  color: var(--tc-subtle);
  font-size: 6px;
  font-weight: 750;
  text-transform: uppercase;
}

.tc-provider-mark,
.tc-model-symbol {
  display: grid;
  flex: none;
  border-radius: 7px;
  color: #fff;
  font-family: Georgia, "Times New Roman", serif;
  font-weight: 700;
  place-items: center;
}

.tc-provider-mark {
  width: 28px;
  height: 28px;
  font-size: 12px;
}

.tc-provider-claude,
.tc-symbol-claude { background: #c77642; }
.tc-provider-gpt,
.tc-symbol-gpt { background: #288066; }
.tc-symbol-gemini { background: #4578c8; }
.tc-symbol-antigravity { background: #bd4f72; }

.tc-service-chip > i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--tc-teal);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--tc-teal) 14%, transparent);
}

.tc-service-claude {
  top: 21px;
  left: 0;
}

.tc-service-gpt {
  right: -2px;
  bottom: 43px;
  animation-delay: -2.5s;
}

.tc-route-pulse {
  top: 46px;
  right: 2px;
  gap: 9px;
  padding: 10px 12px 10px 10px;
  animation: tc-route-in 550ms ease both 1.6s, tc-float 5.5s ease-in-out infinite 2.2s;
}

.tc-route-pulse > span:first-child {
  width: 30px;
  height: 30px;
  display: grid;
  border-radius: 7px;
  color: var(--tc-teal);
  background: var(--tc-teal-soft);
  place-items: center;
}

.tc-route-pulse strong {
  display: block;
  color: var(--tc-teal);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 10px;
}

.tc-model-strip {
  position: relative;
  z-index: 2;
  width: min(1180px, calc(100% - 48px));
  min-height: 104px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 1fr auto 1fr auto 1fr auto 1fr;
  align-items: center;
  border-top: 1px solid var(--tc-line);
  border-bottom: 1px solid var(--tc-line);
}

.tc-model-strip > div {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 9px;
}

.tc-model-strip > i {
  width: 1px;
  height: 27px;
  background: var(--tc-line);
}

.tc-model-symbol {
  width: 27px;
  height: 27px;
  font-size: 10px;
}

.tc-model-strip strong {
  color: var(--tc-ink);
  font-family: Georgia, "Times New Roman", serif;
  font-size: 15px;
  font-weight: 500;
}

.tc-model-strip small {
  color: var(--tc-subtle);
  font-size: 7px;
  font-weight: 750;
  text-transform: uppercase;
}

.tc-capabilities {
  width: min(1180px, calc(100% - 48px));
  margin: 0 auto;
  padding: 112px 0 120px;
}

.tc-section-heading {
  display: grid;
  grid-template-columns: 1.05fr 0.95fr;
  align-items: end;
  gap: 18px 76px;
  margin-bottom: 43px;
}

.tc-section-heading > p,
.tc-developer-copy > p {
  grid-column: 1 / -1;
  margin: 0;
  color: var(--tc-brand);
  font-size: 9px;
  font-weight: 800;
  text-transform: uppercase;
}

.tc-section-heading h2,
.tc-developer-copy h2 {
  margin: 0;
  color: var(--tc-ink);
  font-family: Georgia, "Times New Roman", serif;
  font-size: 47px;
  font-weight: 400;
  line-height: 1.08;
}

.tc-section-heading > span,
.tc-developer-copy > span {
  color: var(--tc-muted);
  font-size: 14px;
  line-height: 1.75;
}

.tc-feature-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14px;
}

.tc-feature-grid article {
  min-height: 275px;
  padding: 25px 26px;
  border: 1px solid var(--tc-line);
  border-radius: 8px;
  background: color-mix(in srgb, var(--tc-surface) 58%, transparent);
  transition: transform 200ms ease, border-color 200ms ease, background 200ms ease, box-shadow 200ms ease;
}

.tc-feature-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 58px;
}

.tc-feature-top > span {
  width: 42px;
  height: 42px;
  display: grid;
  border: 1px solid color-mix(in srgb, var(--tc-brand) 24%, transparent);
  border-radius: 8px;
  color: var(--tc-brand);
  background: color-mix(in srgb, var(--tc-brand) 9%, transparent);
  place-items: center;
}

.tc-feature-top small {
  color: var(--tc-subtle);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 9px;
}

.tc-feature-grid h3 {
  margin: 0 0 10px;
  color: var(--tc-ink);
  font-family: Georgia, "Times New Roman", serif;
  font-size: 22px;
  font-weight: 500;
}

.tc-feature-grid p {
  margin: 0;
  color: var(--tc-muted);
  font-size: 12px;
  line-height: 1.7;
}

.tc-developer-band {
  width: min(1180px, calc(100% - 48px));
  margin: 0 auto 110px;
  display: grid;
  grid-template-columns: 1fr 0.9fr;
  align-items: center;
  gap: 76px;
  padding: 64px 70px;
  border: 1px solid var(--tc-line);
  background: var(--tc-paper-strong);
}

.tc-developer-copy > p {
  margin-bottom: 14px;
}

.tc-developer-copy h2 {
  margin-bottom: 18px;
}

.tc-terminal-card {
  overflow: hidden;
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 8px;
  color: #ddd8cf;
  background: #2a2b28;
  box-shadow: 0 22px 38px rgba(33, 29, 25, 0.17);
}

.tc-terminal-bar {
  height: 36px;
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 0 13px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.07);
  background: #30312e;
}

.tc-terminal-bar span {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #686962;
}

.tc-terminal-bar small {
  margin-left: auto;
  color: #7e8078;
  font-family: ui-monospace, monospace;
  font-size: 8px;
}

.tc-terminal-card code {
  display: block;
  overflow: hidden;
  padding: 23px 21px 17px;
  color: #f3eee5;
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tc-terminal-card code i {
  margin-right: 8px;
  color: #d5a27d;
  font-style: normal;
}

.tc-terminal-card p {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  padding: 6px 21px;
  color: #aaa9a3;
  font-family: ui-monospace, monospace;
  font-size: 9px;
}

.tc-terminal-card p svg {
  color: #7fb091;
}

.tc-terminal-card > strong {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 12px 13px 13px;
  padding: 10px 9px;
  border: 1px solid rgba(112, 165, 129, 0.15);
  border-radius: 7px;
  color: #acd0b7;
  background: rgba(94, 139, 108, 0.09);
  font-family: ui-monospace, monospace;
  font-size: 9px;
  font-weight: 500;
}

.tc-terminal-card > strong i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #78a987;
  animation: tc-status-pulse 2s ease-in-out infinite;
}

.tc-footer {
  width: min(1180px, calc(100% - 48px));
  min-height: 82px;
  margin: 0 auto;
  display: grid;
  grid-template-columns: 1fr 2fr 1fr;
  align-items: center;
  border-top: 1px solid var(--tc-line);
  color: var(--tc-subtle);
  font-size: 9px;
}

.tc-footer p {
  text-align: center;
}

.tc-footer > a {
  justify-self: end;
}

.tc-footer-mark {
  font-size: 15px;
}

.tc-footer-mark img {
  width: 24px;
  height: 24px;
}

@media (hover: hover) {
  .tc-nav-links a:hover,
  .tc-footer a:hover {
    color: var(--tc-ink);
  }

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

  .tc-secondary-action:hover {
    border-color: color-mix(in srgb, var(--tc-brand) 40%, transparent);
    background: var(--tc-surface);
    transform: translateY(-1px);
  }

  .tc-gateway-card:hover {
    box-shadow: 0 36px 74px rgba(58, 47, 37, 0.16), 0 6px 18px rgba(58, 47, 37, 0.08);
    transform: rotateY(0) rotateX(0) translateY(-2px);
  }

  .tc-feature-grid article:hover {
    border-color: color-mix(in srgb, var(--tc-brand) 30%, transparent);
    background: color-mix(in srgb, var(--tc-surface) 88%, transparent);
    box-shadow: 0 12px 28px rgba(55, 46, 39, 0.06);
    transform: translateY(-2px);
  }
}

.tc-icon-button:active,
.tc-login-button:active,
.tc-primary-action:active,
.tc-secondary-action:active {
  transform: scale(0.97);
}

.tc-icon-button:focus-visible,
.tc-login-button:focus-visible,
.tc-primary-action:focus-visible,
.tc-secondary-action:focus-visible,
.tc-nav-links a:focus-visible,
.tc-footer a:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--tc-brand) 65%, transparent);
  outline-offset: 3px;
}

@keyframes tc-status-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.55; transform: scale(0.8); }
}

@keyframes tc-draw-line {
  from { stroke-dashoffset: 700; }
  to { stroke-dashoffset: 0; }
}

@keyframes tc-point-pulse {
  0% { opacity: 0.9; transform: scale(0.65); }
  75%, 100% { opacity: 0; transform: scale(1.7); }
}

@keyframes tc-request-in {
  from { opacity: 0; transform: translateY(7px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes tc-float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-5px); }
}

@keyframes tc-route-in {
  from { opacity: 0; transform: translateX(10px); }
  to { opacity: 1; transform: translateX(0); }
}

@media (max-width: 980px) {
  .tc-nav-links {
    display: none;
  }

  .tc-hero {
    grid-template-columns: 1fr;
    gap: 20px;
    padding-top: 72px;
  }

  .tc-hero-copy {
    max-width: 680px;
    margin: 0 auto;
    text-align: center;
  }

  .tc-live-badge,
  .tc-hero-description {
    margin-left: auto;
    margin-right: auto;
  }

  .tc-hero-actions,
  .tc-proof-row {
    justify-content: center;
  }

  .tc-hero-visual {
    width: min(640px, 100%);
    margin: 0 auto;
  }

  .tc-model-strip {
    grid-template-columns: 1fr 1fr;
    padding: 18px 0;
  }

  .tc-model-strip > i {
    display: none;
  }

  .tc-model-strip > div {
    min-height: 55px;
  }

  .tc-section-heading {
    grid-template-columns: 1fr;
  }

  .tc-developer-band {
    gap: 42px;
    padding: 54px 46px;
  }
}

@media (max-width: 740px) {
  .tc-nav,
  .tc-hero,
  .tc-model-strip,
  .tc-capabilities,
  .tc-developer-band,
  .tc-footer {
    width: min(100% - 32px, 560px);
  }

  .tc-nav {
    height: 60px;
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
    min-height: auto;
    padding: 58px 0 48px;
  }

  .tc-hero h1 {
    font-size: 50px;
  }

  .tc-hero h2 {
    font-size: 38px;
  }

  .tc-hero-description {
    font-size: 14px;
  }

  .tc-hero-visual {
    min-height: 430px;
  }

  .tc-gateway-card {
    width: 96%;
  }

  .tc-capabilities {
    padding: 88px 0;
  }

  .tc-section-heading h2,
  .tc-developer-copy h2 {
    font-size: 38px;
  }

  .tc-feature-grid {
    grid-template-columns: 1fr;
  }

  .tc-feature-grid article {
    min-height: 238px;
  }

  .tc-feature-top {
    margin-bottom: 36px;
  }

  .tc-developer-band {
    grid-template-columns: 1fr;
    margin-bottom: 76px;
    padding: 40px 24px 24px;
  }

  .tc-footer {
    grid-template-columns: 1fr auto;
    padding: 20px 0;
  }

  .tc-footer p {
    grid-column: 1 / -1;
    grid-row: 2;
    text-align: left;
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

  .tc-hero-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .tc-primary-action,
  .tc-secondary-action {
    width: 100%;
  }

  .tc-proof-row {
    gap: 10px 13px;
  }

  .tc-hero-visual {
    min-height: 390px;
  }

  .tc-gateway-card {
    width: 100%;
    padding: 15px;
  }

  .tc-service-chip {
    width: 132px;
    transform: scale(0.84);
  }

  .tc-service-claude {
    top: 0;
    left: -18px;
  }

  .tc-service-gpt {
    right: -18px;
    bottom: 4px;
  }

  .tc-route-pulse {
    top: 26px;
    right: -13px;
    transform: scale(0.88);
  }

  .tc-route-chart svg {
    height: 88px;
  }

  .tc-model-strip {
    grid-template-columns: 1fr;
  }

  .tc-model-strip > div {
    border-bottom: 1px solid var(--tc-line);
  }

  .tc-model-strip > div:last-child {
    border-bottom: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .tc-home *,
  .tc-home *::before,
  .tc-home *::after {
    scroll-behavior: auto !important;
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
