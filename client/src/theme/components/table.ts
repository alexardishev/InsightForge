import { tableAnatomy } from '@chakra-ui/anatomy';
import { createMultiStyleConfigHelpers } from '@chakra-ui/react';

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(tableAnatomy.keys);

const dataGrid = definePartsStyle({
  table: {
    borderCollapse: 'separate',
    borderSpacing: 0,
  },
  th: {
    bg: 'bg.elevated',
    color: 'text.muted',
    fontWeight: 600,
    letterSpacing: '0.02em',
    textTransform: 'none',
    position: 'sticky',
    top: 0,
    zIndex: 1,
  },
  td: {
    borderBottom: '1px solid',
    borderColor: 'border.subtle',
  },
});

export const Table = defineMultiStyleConfig({
  variants: { dataGrid },
  defaultProps: { variant: 'dataGrid', size: 'md' },
});
