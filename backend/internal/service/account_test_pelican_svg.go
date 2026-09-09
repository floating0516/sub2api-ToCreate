package service

// openAIDrawPelicanBicycleHTML is a self-contained HTML document whose body is
// an animated SVG: a pelican riding a bicycle. CSS/SMIL only, so it can play
// inside a sandboxed iframe without scripts.
const openAIDrawPelicanBicycleHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>鹈鹕骑自行车</title>
<style>
  html, body { margin: 0; height: 100%; background: #d7efff; overflow: hidden; }
  svg { display: block; width: 100%; height: 100%; }
</style>
</head>
<body>
<svg viewBox="0 0 640 360" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="鹈鹕骑自行车的二维动画">
  <defs>
    <linearGradient id="sky" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%" stop-color="#9ad8ff"/>
      <stop offset="70%" stop-color="#e7f6ff"/>
      <stop offset="100%" stop-color="#cfe9c4"/>
    </linearGradient>
    <linearGradient id="road" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0%" stop-color="#8b8f97"/>
      <stop offset="100%" stop-color="#5c616a"/>
    </linearGradient>
  </defs>

  <rect width="640" height="360" fill="url(#sky)"/>
  <ellipse cx="520" cy="58" rx="36" ry="36" fill="#fff6c8"/>
  <g fill="#ffffff">
    <g>
      <ellipse cx="90" cy="70" rx="34" ry="16"/>
      <ellipse cx="118" cy="70" rx="22" ry="12"/>
      <animateTransform attributeName="transform" type="translate" values="0 0; 40 0; 0 0" dur="12s" repeatCount="indefinite"/>
    </g>
    <g>
      <ellipse cx="250" cy="46" rx="40" ry="18"/>
      <ellipse cx="282" cy="46" rx="24" ry="12"/>
      <animateTransform attributeName="transform" type="translate" values="0 0; -50 0; 0 0" dur="16s" repeatCount="indefinite"/>
    </g>
  </g>
  <ellipse cx="80" cy="250" rx="90" ry="28" fill="#9ecb7a"/>
  <ellipse cx="560" cy="255" rx="110" ry="32" fill="#8fbe6d"/>
  <rect x="0" y="268" width="640" height="92" fill="url(#road)"/>
  <g stroke="#f4d35e" stroke-width="8" stroke-dasharray="28 22">
    <line x1="-80" y1="312" x2="720" y2="312">
      <animate attributeName="x1" values="-80; 20; -80" dur="1.2s" repeatCount="indefinite"/>
      <animate attributeName="x2" values="720; 820; 720" dur="1.2s" repeatCount="indefinite"/>
    </line>
  </g>

  <g>
    <animateTransform attributeName="transform" type="translate" values="-80 0; 720 0" dur="7s" repeatCount="indefinite"/>
    <g transform="translate(0 18)">
      <g transform="translate(0 0)">
        <animateTransform attributeName="transform" type="translate" values="0 0; 0 -4; 0 0" dur="0.45s" repeatCount="indefinite"/>

        <!-- bicycle -->
        <g fill="none" stroke="#2c3340" stroke-width="5" stroke-linecap="round" stroke-linejoin="round">
          <g transform="translate(78 214)">
            <circle r="28" fill="#f7f7f7"/>
            <circle r="28"/>
            <circle r="5" fill="#2c3340"/>
            <g>
              <line x1="0" y1="0" x2="0" y2="-24"/>
              <line x1="0" y1="0" x2="21" y2="12"/>
              <line x1="0" y1="0" x2="-21" y2="12"/>
              <animateTransform attributeName="transform" type="rotate" from="0" to="360" dur="0.7s" repeatCount="indefinite"/>
            </g>
          </g>
          <g transform="translate(168 214)">
            <circle r="28" fill="#f7f7f7"/>
            <circle r="28"/>
            <circle r="5" fill="#2c3340"/>
            <g>
              <line x1="0" y1="0" x2="0" y2="-24"/>
              <line x1="0" y1="0" x2="21" y2="12"/>
              <line x1="0" y1="0" x2="-21" y2="12"/>
              <animateTransform attributeName="transform" type="rotate" from="0" to="360" dur="0.7s" repeatCount="indefinite"/>
            </g>
          </g>
          <path d="M78 214 L118 214 L148 168 L168 214"/>
          <path d="M118 214 L132 176 L104 176"/>
          <path d="M148 168 L176 156"/>
          <path d="M104 176 L88 168"/>
        </g>
        <ellipse cx="108" cy="172" rx="16" ry="6" fill="#3d4654"/>

        <!-- pelican -->
        <g transform="translate(96 108)">
          <ellipse cx="38" cy="42" rx="36" ry="22" fill="#f4f1ea"/>
          <path d="M8 40 q18 28 56 8 q-8 22 -40 18 q-28 -4 -16 -26z" fill="#f7e38a"/>
          <circle cx="62" cy="28" r="16" fill="#f4f1ea"/>
          <path d="M74 30 q28 4 36 16 q-22 8 -40 2z" fill="#f0b429"/>
          <path d="M74 32 q24 10 32 22 q-20 2 -34 -8z" fill="#e0891a"/>
          <circle cx="68" cy="24" r="3.2" fill="#2c3340"/>
          <path d="M18 28 q-22 -18 -8 -34 q18 8 22 22z" fill="#e7e2d6">
            <animateTransform attributeName="transform" type="rotate" values="-8 18 28; 16 18 28; -8 18 28" dur="0.5s" repeatCount="indefinite"/>
          </path>
          <path d="M28 58 q8 18 2 28" stroke="#2c3340" stroke-width="4" fill="none" stroke-linecap="round">
            <animate attributeName="d" values="M28 58 q8 18 2 28; M28 58 q-6 16 10 26; M28 58 q8 18 2 28" dur="0.45s" repeatCount="indefinite"/>
          </path>
          <path d="M48 58 q10 16 18 24" stroke="#2c3340" stroke-width="4" fill="none" stroke-linecap="round">
            <animate attributeName="d" values="M48 58 q10 16 18 24; M48 58 q-4 14 8 26; M48 58 q10 16 18 24" dur="0.45s" repeatCount="indefinite"/>
          </path>
        </g>
      </g>
    </g>
  </g>
</svg>
</body>
</html>
`
