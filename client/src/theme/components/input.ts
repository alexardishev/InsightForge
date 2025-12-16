import { defineStyleConfig } from '@chakra-ui/react';

export const Input = defineStyleConfig({
  variants: {
    filledGlass: {
      field: {
        background: 'rgba(255,255,255,0.04)',
        borderColor: 'border.subtle',
        borderWidth: '1px',
        backdropFilter: 'blur(10px)',
        _hover: { borderColor: 'accent.primary' },
        _focusVisible: {
          borderColor: 'accent.primary',
          boxShadow: '0 0 0 1px var(--chakra-colors-accent-primary)',
        },
      },
    },
    outlineSoft: {
      field: {
        borderColor: 'border.subtle',
        _hover: { borderColor: 'accent.primary' },
        _focusVisible: {
          borderColor: 'accent.primary',
          boxShadow: '0 0 0 1px var(--chakra-colors-accent-primary)',
        },
      },
    },
  },
  defaultProps: {
    variant: 'filledGlass',
    size: 'md',
  },
});
