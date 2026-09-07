/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Homepage cream / bronze tokens, shared by the logged-in shell
        paper: {
          DEFAULT: '#f6f4ef',
          surface: '#fffefb'
        },
        // 主色调 - bronze to match the landing page
        primary: {
          50: '#f8f1ea',
          100: '#f0e0d0',
          200: '#e2c4a8',
          300: '#d09a71',
          400: '#c08a60',
          500: '#aa7149',
          600: '#895634',
          700: '#6f4529',
          800: '#54341f',
          900: '#3d2618',
          950: '#24160d'
        },
        // Warm ink / paper neutrals
        gray: {
          50: '#f6f4ef',
          100: '#eeebe3',
          200: '#e0dcd2',
          300: '#c8c3b8',
          400: '#9b978c',
          500: '#6f726c',
          600: '#555850',
          700: '#3e413b',
          800: '#2c2e29',
          900: '#262823',
          950: '#171916'
        },
        // 辅助色 - warm slate
        accent: {
          50: '#f6f4ef',
          100: '#eeebe3',
          200: '#e0dcd2',
          300: '#c8c3b8',
          400: '#9b978c',
          500: '#6f726c',
          600: '#555850',
          700: '#3e413b',
          800: '#2c2e29',
          900: '#262823',
          950: '#171916'
        },
        // 深色模式背景
        dark: {
          50: '#f1eee7',
          100: '#e4e1d8',
          200: '#c8c6bc',
          300: '#b6b7b0',
          400: '#858981',
          500: '#6a6d66',
          600: '#4a4d46',
          700: '#343730',
          800: '#232620',
          900: '#1b1d19',
          950: '#171916'
        }
      },
      fontFamily: {
        serif: ['Georgia', 'Times New Roman', 'serif'],
        sans: [
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
        glass: '0 8px 32px rgba(0, 0, 0, 0.08)',
        'glass-sm': '0 4px 16px rgba(0, 0, 0, 0.06)',
        glow: '0 0 20px rgba(170, 113, 73, 0.22)',
        'glow-lg': '0 0 40px rgba(170, 113, 73, 0.3)',
        card: '0 1px 3px rgba(58, 47, 37, 0.04), 0 1px 2px rgba(58, 47, 37, 0.06)',
        'card-hover': '0 10px 40px rgba(58, 47, 37, 0.08)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.1)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #aa7149 0%, #895634 100%)',
        'gradient-dark': 'linear-gradient(135deg, #232620 0%, #171916 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,254,251,0.1) 0%, rgba(255,254,251,0.05) 100%)',
        'mesh-gradient':
          'radial-gradient(at 40% 20%, rgba(170, 113, 73, 0.1) 0px, transparent 50%), radial-gradient(at 80% 0%, rgba(196, 149, 106, 0.08) 0px, transparent 50%), radial-gradient(at 0% 50%, rgba(137, 86, 52, 0.06) 0px, transparent 50%)'
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
          '0%': { boxShadow: '0 0 20px rgba(170, 113, 73, 0.22)' },
          '100%': { boxShadow: '0 0 30px rgba(170, 113, 73, 0.36)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}
