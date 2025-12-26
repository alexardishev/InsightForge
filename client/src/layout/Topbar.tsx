import React, { useMemo } from 'react';
import {
  Box,
  Flex,
  HStack,
  IconButton,
  InputGroup,
  InputLeftElement,
  Input,
  Text,
  Avatar,
  Badge,
  useColorModeValue,
} from '@chakra-ui/react';
import { FiMenu, FiSearch, FiWifi } from 'react-icons/fi';
import { useSelector } from 'react-redux';
import type { RootState } from '../app/store';
import ThemeToggle from '../components/ThemeToggle';
import { flattenSelections } from '../features/viewBuilder/viewBuilderSlice';

interface TopbarProps {
  onOpenMenu: () => void;
}

const Topbar: React.FC<TopbarProps> = ({ onOpenMenu }) => {
  const builder = useSelector((state: RootState) => state.viewBuilder);
  const selectedSources = useMemo(() => flattenSelections(builder), [builder]);
  const { viewName } = builder;
  const background = useColorModeValue('white', 'bg.surface');

  return (
    <Box
      px={{ base: 4, lg: 6, xl: 10 }}
      py={4}
      borderBottom="1px solid"
      borderColor="border.subtle"
      bg={background}
      backdropFilter="blur(10px)"
      position="sticky"
      top={0}
      zIndex={10}
    >
      <Flex align="center" justify="space-between" gap={4}>
        <HStack spacing={3} align="center">
          <IconButton
            aria-label="Open navigation"
            icon={<FiMenu />}
            variant="ghost"
            display={{ base: 'inline-flex', lg: 'none' }}
            onClick={onOpenMenu}
          />
          <Box>
            <Text fontWeight="bold" fontSize="lg">
              {viewName || 'Data cockpit'}
            </Text>
            <HStack spacing={2} color="text.muted" fontSize="sm" flexWrap="wrap">
              {selectedSources.length === 0 && (
                <Badge colorScheme="gray">Источники не выбраны</Badge>
              )}
              {selectedSources.map((source) => (
                <Badge key={`${source.db}.${source.schema}`} colorScheme="cyan">
                  {source.db}.{source.schema}
                </Badge>
              ))}
            </HStack>
          </Box>
        </HStack>
        <HStack spacing={3} flex="1" maxW="600px" justify="flex-end">
          <InputGroup maxW="360px" display={{ base: 'none', md: 'flex' }}>
            <InputLeftElement pointerEvents="none" color="text.muted">
              <FiSearch />
            </InputLeftElement>
            <Input placeholder="Search tables, columns, views" variant="filledGlass" />
          </InputGroup>
          <Badge
            colorScheme="green"
            variant="subtle"
            display="inline-flex"
            alignItems="center"
            gap={1}
          >
            <FiWifi /> Stable
          </Badge>
          <ThemeToggle />
          <Avatar size="sm" name="Analyst" bg="accent.secondary" color="white" />
        </HStack>
      </Flex>
    </Box>
  );
};

export default Topbar;
