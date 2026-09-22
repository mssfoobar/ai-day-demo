import { defineConfig } from 'unocss'

export default defineConfig({
  theme: {
    colors: {
      // Light canvas
      ink: '#18181b',
      muted: '#71717a',
      faint: '#a1a1aa',
      rule: '#e4e4e7',
      steel: '#d4d4d8',
      paper: '#fafafa',
      shell: '#f4f4f5',

      // Red accent
      brand: {
        DEFAULT: '#dc2626',
        dark: '#b91c1c',
        mute: '#9f5f5f',
        pale: '#d4a0a0',
        wash: 'rgba(220,38,38,0.08)',
        line: 'rgba(220,38,38,0.35)',
      },

      // Terminal panels
      term: {
        DEFAULT: '#18181b',
        fg: '#fafafa',
        dim: '#71717a',
        dimmer: '#52525b',
        ok: '#4ade80',
      },

      claude: '#d97757',

      // Rendered markdown inside spec cards
      code: '#3f3f46',
      syntax: '#c4c4c8',
    },
  },
  // Slide 1 renders before on-demand scanning reaches its components during export.
  safelist: [
    'deck-row', 'py-2.5', 'w-10', 'text-brand', 'text-xl', 'text-ink', 'mt-8', 'text-left',
    // Points and NumStep, which open the deck on the contents slide.
    'w-full', 'mx-auto', 'font-mono', 'text-base', 'py-1', 'pl-10', 'leading-relaxed', 'text-muted',
    // SectionTitle, which can open the deck.
    'flex', 'flex-col', 'items-center', 'justify-center', 'text-6xl', 'font-extrabold',
    'tracking-tight', 'mt-6', 'text-center',
    // Click reveals toggle these; generating them on demand re-renders the slide mid-click.
    'opacity-0', 'opacity-100', 'pt-2.5',
  ],
  shortcuts: {
    'deck-panel': 'rounded-xl px-8 py-6 text-left bg-term',
    'deck-mono': 'font-mono text-[13px] leading-relaxed',
    'deck-row': 'flex items-baseline',
  },
})
