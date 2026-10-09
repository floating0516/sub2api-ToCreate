# ToCreate independent preview

Baseline: production commit ddc9be1e8343335b57dd2df232d447557097b59b.
Production routes, authentication components, APIs and deployment are untouched.
Only build/test in GitHub Actions; do not install or compile on the production VPS.

The homepage retains the accepted simplified warm ToCreate design. Authentication
now uses the original AuthLayout / EmailFirstAuthDialog visual language and email-
first flow. The experimental compact tab bar, three-row form and empty row are no
longer mounted. The original existing gradients, fonts, sizes, layout and colors
are preserved rather than replaced with a new design.

OriginalAuthPreview.vue is a UI-only adaptation of the original markup; the
original-auth.css snapshot comes directly from the baseline components' scoped
styles, with global/deep selectors resolved for this independent entry.
original-auth-motion.css contains the preview reset/icon sizing, demo toolbar,
mobile input font safety and the motion layer. It adds button release springs,
finite outgoing/incoming transitions and in-place loading feedback. Resting
geometry, color and typography are checked with and without added motion.

The default email entry precedes login (one password field) or registration
(password and confirmation). Verification uses a six-digit sample code. No
three-row layout is introduced. A 370px stage reserves the original registration height using 30px from the empty
footer area; short viewports reclaim another 34px of footer margins. Controls and
brand sizes stay original. Only genuinely larger content resizes. Extremely short windows retain
natural scrolling instead of hiding fields or compressing the original design.

?view=login or ?view=register selects the entry intent; ?lang=en selects English.
&still=1 turns off motion for visual comparison. “Use sample details” supplies
hello@example.com / DemoOnly2026! / 123456. All submissions are simulated and
explicitly labeled. No API clients, emails, real accounts or persistent storage.
Recovery and success explain their demo status. OAuth/CAPTCHA/backend options are
outside this visual preview; no production authentication rules are changed.

The workflow validates compilation, typing and existing unit tests, then records
real desktop mouse / simulated mobile touch flows. It checks narrow bilingual
layouts, static visual parity, email-first steps, keyboard focus, invalid input,
repeated operations, stale callbacks, reduced motion and unmount cleanup.
