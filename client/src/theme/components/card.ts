import { cardAnatomy } from '@chakra-ui/anatomy';
import { createMultiStyleConfigHelpers } from '@chakra-ui/react';

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(cardAnatomy.keys);

const surface = definePartsStyle({
  container: {
    background: 'bg.surface',
    border: '1px solid',
    borderColor: 'border.subtle',
    borderRadius: 'lg',
    boxShadow: 'subtle',
  },
});

const glass = definePartsStyle({
  container: {
    background: 'bg.elevated',
    border: '1px solid',
    borderColor: 'border.strong',
    backdropFilter: 'blur(12px)',
    boxShadow: 'subtle',
  },
});

const interactive = definePartsStyle({
  container: {
    background: 'bg.surface',
    border: '1px solid',
    borderColor: 'border.strong',
    transition: 'all 150ms ease',
    _hover: { borderColor: 'accent.primary', boxShadow: 'glow' },
  },
});

export const Card = defineMultiStyleConfig({
  variants: { surface, glass, interactive },
  defaultProps: { variant: 'surface' },
});
