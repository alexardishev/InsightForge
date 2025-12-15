import { extendTheme, type ThemeConfig } from '@chakra-ui/react';
import type { StyleFunctionProps } from '@chakra-ui/styled-system';
import { fonts, fontSizes, radii, shadows, semanticTokens, config as colorModeConfig } from './tokens';
import { Button } from './components/button';
import { Input } from './components/input';
import { Card } from './components/card';
import { Table } from './components/table';

const config: ThemeConfig = {
  ...colorModeConfig,
};

export const theme = extendTheme({
  config,
  fonts,
  fontSizes,
  radii,
  shadows,
  semanticTokens,
  styles: {
    global: ({ colorMode }: StyleFunctionProps) => ({
      'html, body': {
        background: 'bg.canvas',
        color: 'text.primary',
        minHeight: '100%',
      },
      '*::selection': {
        background: colorMode === 'dark' ? 'accent.secondary' : 'accent.primary',
        color: 'white',
      },
    }),
  },
  components: {
    Button,
    Input,
    Card,
    Table,
    Modal: {
      baseStyle: {
        dialog: {
          bg: 'bg.surface',
          border: '1px solid',
          borderColor: 'border.strong',
          borderRadius: 'xl',
        },
      },
    },
    Tabs: {
      baseStyle: {
        tab: {
          fontWeight: 600,
          borderRadius: 'md',
          _selected: {
            color: 'accent.primary',
            boxShadow: '0 1px 0 0 var(--chakra-colors-accent-primary)',
          },
        },
      },
    },
    Badge: {
      baseStyle: {
        borderRadius: 'full',
        textTransform: 'none',
        fontWeight: 600,
      },
    },
    Tooltip: {
      baseStyle: {
        bg: 'bg.elevated',
        border: '1px solid',
        borderColor: 'border.strong',
        color: 'text.primary',
      },
    },
  },
});

export default theme;
