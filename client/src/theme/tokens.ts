import type { ThemeConfig } from '@chakra-ui/react';

export const fonts = {
  heading: 'Inter, "Manrope", system-ui, -apple-system, sans-serif',
  body: 'Inter, "Manrope", system-ui, -apple-system, sans-serif',
  mono: '"JetBrains Mono", Menlo, monospace',
};

export const fontSizes = {
  xs: '12px',
  sm: '14px',
  md: '16px',
  lg: '18px',
  xl: '22px',
  '2xl': '28px',
};

export const radii = {
  none: '0',
  sm: '8px',
  md: '10px',
  lg: '12px',
  xl: '16px',
  full: '9999px',
};

export const shadows = {
  subtle: '0 10px 40px rgba(0, 0, 0, 0.18)',
  glow: '0 10px 40px rgba(56, 189, 248, 0.35)',
};

export const semanticTokens = {
  colors: {
    'bg.canvas': {
      default: '#0f1628',
      _light: '#eef2f7',
    },
    'bg.surface': {
      default: 'rgba(25, 35, 58, 0.92)',
      _light: '#ffffff',
    },
    'bg.elevated': {
      default: 'rgba(34, 48, 80, 0.9)',
      _light: '#f6f8fc',
    },
    'text.primary': {
      default: '#f8fbff',
      _light: '#0f172a',
    },
    'text.muted': {
      default: '#b5c4e1',
      _light: '#4b5563',
    },
    'accent.primary': {
      default: '#7ad7f0',
      _light: '#22b8e9',
    },
    'accent.secondary': {
      default: '#c8b5ff',
      _light: '#8f7cf3',
    },
    'border.subtle': {
      default: 'rgba(255, 255, 255, 0.08)',
      _light: 'rgba(15, 23, 42, 0.08)',
    },
    'border.strong': {
      default: 'rgba(255, 255, 255, 0.18)',
      _light: 'rgba(15, 23, 42, 0.18)',
    },
    'status.success': {
      default: '#34d399',
      _light: '#16a34a',
    },
    'status.warning': {
      default: '#fbbf24',
      _light: '#d97706',
    },
    'status.error': {
      default: '#f87171',
      _light: '#dc2626',
    },
    'status.info': {
      default: '#38bdf8',
      _light: '#0ea5e9',
    },
  },
};

export const config: ThemeConfig = {
  initialColorMode: 'dark',
  useSystemColorMode: false,
};
