# ToCreate simple homepage preview

Independent Vite entry based on production commit
`ddc9be1e8343335b57dd2df232d447557097b59b`. The production homepage and router
remain untouched. Build only in GitHub Actions; do not install or compile on the VPS.

Current direction: a warm neutral canvas, Georgia brand heading, one primary
Get started link and a header Login link. No model card, simulated requests,
model names, marketing descriptions or feature cards are mounted. The previous
ModelGatewayDemo source is retained as history but is not imported by this entry.
Both entry links go directly to https://api.lihe.chat/login using normal anchors;
there is no fake loading state or navigation delay. Native modifier-click works.

EntryLink.vue reuses the existing Icon component. Shared button feedback uses
180–240ms CSS transitions for corner radius, arrow movement, background and
pressing. Hover is gated to fine pointers, touch has active feedback, and keyboard
focus is visible. Reduced motion removes transitions and positional transforms.
No animation dependencies, timers, request clients or infinite animations.

The responsive layout uses svh and intrinsic minimum height so short screens can
scroll naturally. English and Chinese share the existing vue-i18n setup; ?lang=en
selects English. Copy lives in home.simplePreview in both landing dictionaries.

The preview workflow lints, type-checks, checks locale completeness, builds, then
runs Playwright on remote CI. verify-record.mjs covers bilingual narrow layouts,
keyboard navigation, touch, reduced motion, repeated activation and stable layout.
Desktop/mobile recordings include hover, pressing, language switch and keyboard
focus. Only the test intercepts outbound login navigation to keep recording the
preview; actual preview links navigate normally. MP4 and WebM are published as
evidence. No live authentication/API requests are made by the test.
