/** @type {import('tailwindcss').Config} */
const oracleOrange = {
  50: '#fff4ed',
  100: '#ffe2d3',
  200: '#ffc4a8',
  300: '#ffa079',
  400: '#ff8152',
  500: '#ff6b35',
  600: '#e95220',
  700: '#bd3d16',
  800: '#963219',
  900: '#7a2d1a',
  950: '#42150a'
}

const oracleYellow = {
  50: '#fffceb',
  100: '#fff7c2',
  200: '#ffed85',
  300: '#ffe05c',
  400: '#ffd746',
  500: '#ffd23f',
  600: '#d9a900',
  700: '#a87900',
  800: '#815c08',
  900: '#694b0c',
  950: '#3d2804'
}

const oracleBlue = {
  50: '#eef7ff',
  100: '#d8ecff',
  200: '#b9dcff',
  300: '#89c5ff',
  400: '#50a4f5',
  500: '#1e7fd0',
  600: '#075fae',
  700: '#004e98',
  800: '#073f76',
  900: '#0b365f',
  950: '#07213d'
}

const oracleGreen = {
  50: '#effaf1',
  100: '#dff3df',
  200: '#bfe7c4',
  300: '#91d49e',
  400: '#5dbd72',
  500: '#2d9b4e',
  600: '#247f40',
  700: '#206536',
  800: '#1c512e',
  900: '#174326',
  950: '#0b2514'
}

const oracleRed = {
  50: '#fff1f2',
  100: '#ffe0e3',
  200: '#ffc6cc',
  300: '#ff9aa6',
  400: '#f46170',
  500: '#e63946',
  600: '#c62433',
  700: '#a51e2c',
  800: '#891e29',
  900: '#741f28',
  950: '#400b10'
}

const oracleGray = {
  50: '#fffef8',
  100: '#fff8e3',
  200: '#eadfbd',
  300: '#d1c49e',
  400: '#9b9077',
  500: '#706757',
  600: '#50483f',
  700: '#393137',
  800: '#342c3a',
  900: '#2b2533',
  950: '#17121d'
}

export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        white: '#fffef8',
        black: '#2b2533',
        cream: '#fffdf4',
        ink: '#2b2533',
        nb: {
          cream: '#fffdf4',
          ink: '#2b2533',
          orange: '#ff6b35',
          yellow: '#ffd23f',
          blue: '#004e98',
          green: '#2d9b4e',
          red: '#e63946'
        },
        gray: oracleGray,
        slate: oracleGray,
        zinc: oracleGray,
        neutral: oracleGray,
        stone: oracleGray,
        // Oracle UI semantic palettes.
        primary: oracleOrange,
        accent: oracleBlue,
        orange: oracleOrange,
        amber: oracleYellow,
        yellow: oracleYellow,
        blue: oracleBlue,
        sky: oracleBlue,
        cyan: oracleBlue,
        indigo: oracleBlue,
        purple: oracleBlue,
        pink: oracleRed,
        rose: oracleRed,
        red: oracleRed,
        green: oracleGreen,
        emerald: oracleGreen,
        lime: oracleGreen,
        teal: oracleGreen,
        violet: oracleBlue,
        fuchsia: oracleRed,
        // Optional dark mode keeps the same ink-and-paper visual language.
        dark: {
          50: '#fffef8',
          100: '#f4edd6',
          200: '#ded3b5',
          300: '#c4b791',
          400: '#9b8e75',
          500: '#716553',
          600: '#55485e',
          700: '#3a3042',
          800: '#302737',
          900: '#211a27',
          950: '#17121d'
        }
      },
      fontFamily: {
        sans: [
          'Sora',
          'Noto Sans SC',
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        xs: '1px 1px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        sm: '1.5px 1.5px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        DEFAULT: '1.5px 1.5px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        md: '2px 2px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        lg: '3px 3px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        xl: '3px 3px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        '2xl': '4px 4px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        inner: 'inset 0 0 0 1px var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        none: 'none',
        glass: '3px 3px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        'glass-sm': '1.5px 1.5px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        glow: '2px 2px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        'glow-lg': '3px 3px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        card: '3px 3px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        'card-hover': '4px 4px 0 var(--nb-shadow-color, rgb(43 37 51 / 0.62))',
        'inner-glow': 'inset 0 0 0 1px var(--nb-shadow-color, rgb(43 37 51 / 0.62))'
      },
      backgroundImage: {
        'gradient-to-t':
          'linear-gradient(to top, var(--tw-gradient-from), var(--tw-gradient-from))',
        'gradient-to-tr':
          'linear-gradient(to top right, var(--tw-gradient-from), var(--tw-gradient-from))',
        'gradient-to-r':
          'linear-gradient(to right, var(--tw-gradient-from), var(--tw-gradient-from))',
        'gradient-to-br':
          'linear-gradient(to bottom right, var(--tw-gradient-from), var(--tw-gradient-from))',
        'gradient-to-b':
          'linear-gradient(to bottom, var(--tw-gradient-from), var(--tw-gradient-from))',
        'gradient-to-bl':
          'linear-gradient(to bottom left, var(--tw-gradient-from), var(--tw-gradient-from))',
        'gradient-to-l':
          'linear-gradient(to left, var(--tw-gradient-from), var(--tw-gradient-from))',
        'gradient-to-tl':
          'linear-gradient(to top left, var(--tw-gradient-from), var(--tw-gradient-from))',
        'gradient-radial':
          'radial-gradient(circle, var(--tw-gradient-from), var(--tw-gradient-from))',
        'gradient-primary': 'linear-gradient(#ff6b35, #ff6b35)',
        'gradient-dark': 'linear-gradient(#211a27, #211a27)',
        'gradient-glass': 'linear-gradient(#fffef8, #fffef8)',
        'mesh-gradient': 'none'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 20px rgba(20, 184, 166, 0.25)' },
          '100%': { boxShadow: '0 0 30px rgba(20, 184, 166, 0.4)' }
        }
      },
      backdropBlur: {
        none: '0',
        0: '0',
        xs: '0',
        sm: '0',
        DEFAULT: '0',
        md: '0',
        lg: '0',
        xl: '0',
        '2xl': '0',
        '3xl': '0'
      },
      borderRadius: {
        none: '0',
        sm: '4px',
        DEFAULT: '5px',
        md: '6px',
        lg: '6px',
        xl: '8px',
        '2xl': '8px',
        '3xl': '8px',
        '4xl': '8px',
        full: '9999px'
      },
      borderWidth: {
        DEFAULT: '1.5px',
        0: '0',
        2: '2px',
        4: '3px',
        8: '5px'
      },
      letterSpacing: {
        tighter: '0',
        tight: '0',
        normal: '0',
        wide: '0',
        wider: '0',
        widest: '0'
      }
    }
  },
  plugins: []
}
