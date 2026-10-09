# ToCreate model gateway preview

Based on production commit `ddc9be1e8343335b57dd2df232d447557097b59b`.
This independent Vite entry reuses Vue, vue-i18n, the landing locale dictionaries
and the existing Icon component. The production homepage and router are unchanged.

State flow: key → model selection → simulated request → response → success → key.
The response stays until the visitor chooses Done or Try again. Reset is always
available. There are no API clients, real credentials, model calls or audio.

The `.github/workflows/model-gateway-preview.yml` workflow type-checks the preview,
checks spring behavior and locale completeness, then publishes a static artifact.
Do not install dependencies or compile on the production VPS. On a development
machine or CI: `pnpm exec vite build --config vite.preview.config.ts`.

Query parameters: `?lang=en`, `?baseline=1` (original static card in the same
preview layout), `?qa=1` (adds a component mount/unmount control for verification).

Motion: 70ms content exit, 380ms spring geometry, 300ms staggered content entry.
Only the shape's dimensions/radius/color animate; text is never scaled.
The tab edges use separate analytic springs. WAAPI animations terminate at rest.
Visibility and reduced-motion changes finish active transitions; the loading
indicator pauses when hidden. Unmount clears animations, observers and timers.

Round 2: a persistent SVG marker carries the key ring through request progress
into a success check. It sits outside the fading content and has its own geometry
and stroke transitions. The desktop reserved stage is 352px; Chinese mobile is
340px, English 376px for wrapping. Helper text is 12px with darker neutral colors.
Model arrow-key focus uses preventScroll, and the demo excludes its moving
descendants from browser scroll anchoring. No scroll position is forcibly restored,
so the visitor can still scroll freely during a request.

`verify-record.mjs` runs in remote CI with isolated Playwright tooling (no product
dependency). It records uninterrupted, real-time mouse and touch flows, checks
their scroll positions and persistent element identity, and covers bilingual
narrow layouts, keyboard use, cancellation and reduced motion. MP4 and WebM
recordings are published alongside the browser evidence artifact.

Geist and its SIL OFL license are copied from the supplied one-shape reference.
The public site logo is a snapshot of the existing site's public branding asset.
Neither the music nor the fixed playback timeline is included.
