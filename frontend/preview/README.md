# ToCreate simple homepage preview

Independent Vite entry based on production commit
`ddc9be1e8343335b57dd2df232d447557097b59b`. The production homepage and router
remain untouched. Build only in GitHub Actions; do not install or compile on the VPS.

Current direction: the original warm white/brown brand palette, Georgia brand heading, one primary
Get started link and a header Login link. No model card, simulated requests,
model names, marketing descriptions or feature cards are mounted. The previous
ModelGatewayDemo source is retained as history but is not imported by this entry.
Both home entry links now open the isolated login preview using normal anchors
and same-page navigation. Native modifier-click works on these entry links.

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
Desktop/mobile recordings cover the complete login, registration, sample email
verification and recovery flows. MP4 and WebM are published as evidence. Neither
the prototype nor its tests make live authentication/API requests.

## Authentication preview

`?view=login` and `?view=register` now connect to an isolated authentication
prototype. Home entry links stay within the preview; modifier-click and browser
history work. Production authentication components are unchanged.

Login: form → simulated processing → demo completion.
Registration: form → simulated processing → sample verification code → completion.
Recovery: email → simulated processing → explanation of the real reset email step.
No authentication SDK, API requests, persistent storage or emails. “Use sample
details” populates fictional input, and the fixed sample verification code is
123456. The page is explicitly labeled as an interactive preview.

AuthPreview reuses the existing spring/indicator functions and Icon component.
A persistent surface resizes with a 400ms sampled spring, the tab indicator uses
separate leading/trailing springs, and a shared SVG changes lock/envelope/check.
Text exits before entering, using brief blur only during transitions. Submission
feedback lives on the same button. Motion respects reduced-motion and visibility;
Vue transitions are finite, request timers and WAAPI animations are canceled on
unmount. Epoch tokens reject stale simulated requests; switching tabs or canceling
invalidates the active request. Keyboard tab switching preserves scroll position.

Remote browser verification covers registration, login, recovery, invalid inputs,
keyboard focus, real touch/mouse recordings, rapid toggles, cancellation, reduced
motion, narrow bilingual layouts, back/reload and cleanup after leaving the page.
This is design validation; backend-dependent OAuth, CAPTCHA, invitation and
agreement settings still require the later approved production integration.

## Original brand palette restored

Homepage, login, registration and completion now share the original landing
colors: paper #f6f4ef, surface #fffefb, accent #aa7149 and primary #895634.
Focus rings, tabs and shared marks use warm brown accents. Completion uses a
pale warm surface instead of a dark inversion. Geometry and motion are unchanged.

## Stable login / registration layout

The icon and tab switcher now share a row. Both forms reserve three field rows;
the third login slot is an empty CSS grid item, with no hidden input or focus stop.
Submit and cancel share their existing action rows, so a third field or loading
state cannot push the button down. The redundant introductory sentence is omitted
on the form step. The auth page header/footer and short-screen spacing are compact.
Below 650px viewport height the secondary footer is hidden; no page scroll lock or
fixed-height clipping is used. Very short windows, validation messages, text zoom
and software keyboards can still use natural document scrolling.

The browser regression measures page height, card height and submit position for
login and registration at 1280×720, 1366×768, 390×844, 375×667, 360×640 and 320×568
in both languages, including the loading/cancel row. Screenshots cover desktop and
short phones.
