import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        brand: {
          50:  '#f0f4ff',
          100: '#dce7ff',
          200: '#bcd1ff',
          300: '#8bb1ff',
          400: '#5585ff',
          500: '#2e5fff',
          600: '#1a3ef5',
          700: '#132de1',
          800: '#1526b6',
          900: '#162590',
          950: '#111757',
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
} satisfies Config
