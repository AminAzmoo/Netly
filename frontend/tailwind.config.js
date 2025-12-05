/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        neonA: 'var(--color-neon-a)',
        neonB: 'var(--color-neon-b)',
        bgBase: 'var(--color-bg-base)',
        surface: 'var(--color-surface)',
        primary: 'var(--color-neon-a)',
        warning: 'var(--color-warning)',
        error: 'var(--color-error)',
        silver: {
          100: 'var(--color-silver-100)',
          200: 'var(--color-silver-200)',
          300: 'var(--color-silver-300)',
          400: 'var(--color-silver-400)',
          500: 'var(--color-silver-500)',
        },
      },
      backgroundImage: {
        'dot-grid': 'radial-gradient(circle, var(--color-grid-dot) 1px, transparent 1px)',
      },
      backgroundSize: {
        'dot-grid': '20px 20px',
      },
      animation: {
        'pulse-glow': 'pulse-glow 2s ease-in-out infinite',
        'glow-sweep': 'glow-sweep 3s ease-in-out infinite',
      },
      keyframes: {
        'pulse-glow': {
          '0%, 100%': { opacity: '1', transform: 'scale(1)' },
          '50%': { opacity: '0.7', transform: 'scale(1.1)' },
        },
        'glow-sweep': {
          '0%, 100%': { transform: 'translateY(0)', opacity: '0.3' },
          '50%': { transform: 'translateY(100%)', opacity: '0.6' },
        },
      },
    },
  },
  plugins: [],
}
