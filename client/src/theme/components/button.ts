import { defineStyleConfig } from '@chakra-ui/react';

export const Button = defineStyleConfig({
  baseStyle: {
    fontWeight: 600,
    borderRadius: 'md',
    transition: 'all 180ms ease',
    _focusVisible: {
      boxShadow: '0 0 0 3px rgba(63, 225, 247, 0.4)',
    },
  },
  sizes: {
    md: {
      h: 10,
      px: 5,
    },
    lg: {
      h: 12,
      px: 6,
      fontSize: 'md',
    },
  },
  variants: {
    solid: {
      bg: 'accent.primary',
      color: 'black',
      boxShadow: 'glow',
      _hover: { bg: 'accent.primary', transform: 'translateY(-1px)', filter: 'brightness(1.05)' },
      _active: { transform: 'translateY(0)', filter: 'brightness(0.97)' },
    },
    ghost: {
      bg: 'transparent',
      color: 'text.primary',
      _hover: { bg: 'border.subtle' },
      _active: { bg: 'border.strong' },
    },
    outline: {
      borderColor: 'border.strong',
      color: 'text.primary',
      _hover: { borderColor: 'accent.primary', color: 'accent.primary', boxShadow: '0 0 0 1px var(--chakra-colors-accent-primary)' },
    },
    danger: {
      bg: 'status.error',
      color: 'white',
      _hover: { filter: 'brightness(1.05)' },
      _active: { filter: 'brightness(0.95)' },
    },
    glow: {
      bg: 'accent.secondary',
      color: 'white',
      boxShadow: '0 0 24px rgba(168, 85, 247, 0.35)',
      _hover: { filter: 'brightness(1.08)' },
    },
  },
  defaultProps: {
    variant: 'solid',
    size: 'md',
  },
});
