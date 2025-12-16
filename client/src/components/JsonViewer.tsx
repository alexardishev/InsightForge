import React, { useMemo, useState } from 'react';
import {
  Box,
  Code,
  HStack,
  IconButton,
  Stack,
  Text,
  VStack,
  useColorModeValue,
} from '@chakra-ui/react';
import { ChevronDownIcon, ChevronRightIcon } from '@chakra-ui/icons';

interface JsonViewerProps {
  data: unknown;
  label?: string;
  collapsed?: boolean;
}

const isPrimitive = (value: unknown) =>
  value === null || ['string', 'number', 'boolean'].includes(typeof value);

const formatPrimitive = (value: unknown) => {
  if (typeof value === 'string') return `"${value}"`;
  if (value === null) return 'null';
  return String(value);
};

interface NodeProps {
  label?: string;
  value: unknown;
  depth?: number;
  defaultOpen?: boolean;
}

const JsonNode: React.FC<NodeProps> = ({ label, value, depth = 0, defaultOpen }) => {
  const [open, setOpen] = useState(defaultOpen ?? depth === 0);
  const border = useColorModeValue('rgba(255,255,255,0.6)', 'rgba(255,255,255,0.08)');

  const entries = useMemo(() => {
    if (Array.isArray(value)) {
      return value.map((item, index) => ({ key: String(index), val: item }));
    }
    if (value && typeof value === 'object') {
      return Object.entries(value as Record<string, unknown>).map(([key, val]) => ({ key, val }));
    }
    return [];
  }, [value]);

  const isLeaf = isPrimitive(value);

  return (
    <Box
      pl={depth ? 3 : 0}
      borderLeft={depth ? '1px solid' : undefined}
      borderColor={border}
      py={1}
    >
      <HStack spacing={2} align="flex-start">
        {!isLeaf ? (
          <IconButton
            aria-label={open ? 'Collapse' : 'Expand'}
            icon={open ? <ChevronDownIcon /> : <ChevronRightIcon />}
            size="xs"
            variant="ghost"
            onClick={() => setOpen((prev) => !prev)}
          />
        ) : (
          <Box w="24px" />
        )}
        <HStack align="flex-start" spacing={2} flex="1">
          {label && (
            <Text fontWeight="semibold" fontSize="sm" minW="80px" color="text.muted">
              {label}
            </Text>
          )}
          {isLeaf ? (
            <Code whiteSpace="pre" fontSize="sm">
              {formatPrimitive(value)}
            </Code>
          ) : (
            <Text fontSize="sm" color="text.muted">
              {Array.isArray(value)
                ? `Array(${(value as unknown[]).length})`
                : `Object(${entries.length})`}
            </Text>
          )}
        </HStack>
      </HStack>
      {!isLeaf && open && (
        <VStack align="stretch" spacing={1} mt={1}>
          {entries.map(({ key, val }) => (
            <JsonNode key={`${depth}-${key}`} label={key} value={val} depth={depth + 1} />
          ))}
        </VStack>
      )}
    </Box>
  );
};

const JsonViewer: React.FC<JsonViewerProps> = ({ data, label, collapsed }) => {
  return (
    <Stack
      spacing={2}
      p={4}
      borderRadius="lg"
      border="1px solid"
      borderColor="border.subtle"
      bg="bg.elevated"
      boxShadow="md"
    >
      {label && (
        <Text fontWeight="bold" fontSize="md">
          {label}
        </Text>
      )}
      <JsonNode value={data} defaultOpen={!collapsed} />
    </Stack>
  );
};

export default JsonViewer;
