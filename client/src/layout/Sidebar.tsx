import React from 'react';
import {
  Box,
  VStack,
  Text,
  Icon,
  Button,
  useColorModeValue,
  Divider,
  HStack,
  Badge,
  Drawer,
  DrawerOverlay,
  DrawerContent,
} from '@chakra-ui/react';
import { NavLink } from 'react-router-dom';
import {
  FiDatabase,
  FiSettings,
  FiGitBranch,
  FiRepeat,
  FiActivity,
  FiLayers,
  FiZap,
  FiMap,
} from 'react-icons/fi';
import { useSelector } from 'react-redux';
import type { RootState } from '../app/store';

interface SidebarProps {
  isOpen: boolean;
  onClose: () => void;
  isMobile?: boolean;
}

const NavButton: React.FC<{
  label: string;
  path: string;
  icon: any;
  disabled?: boolean;
  badge?: string;
}> = ({ label, path, icon, disabled, badge }) => {
  const activeBg = useColorModeValue('blue.50', 'rgba(63,225,247,0.12)');
  return (
    <Button
      as={NavLink}
      to={path}
      justifyContent="flex-start"
      leftIcon={<Icon as={icon} />}
      variant="ghost"
      height="48px"
      isDisabled={disabled}
      _activeLink={{
        bg: activeBg,
        color: 'accent.primary',
        boxShadow: 'inset 2px 0 0 var(--chakra-colors-accent-primary)',
      }}
    >
      <HStack justify="space-between" w="full">
        <Text fontWeight="semibold">{label}</Text>
        {badge ? (
          <Badge colorScheme="purple" variant="solid">{badge}</Badge>
        ) : null}
      </HStack>
    </Button>
  );
};

const SidebarContent: React.FC = () => {
  const settings = useSelector((state: RootState) => state.settings);
  const builder = useSelector((state: RootState) => state.viewBuilder);

  const selectedColumnsCount = builder.selectedSources.reduce(
    (acc, source) => acc + source.selectedColumns.length,
    0,
  );

  const canBuilder = !!settings.dataBaseInfo;
  const canJoins = selectedColumnsCount > 0;
  const canTransforms = canJoins;
  const canSummary = canTransforms;

  return (
    <Box
      as="nav"
      w={{ base: 'full', lg: '260px' }}
      bg="bg.surface"
      borderRight="1px solid"
      borderColor="border.subtle"
      px={4}
      py={6}
      h="100%"
      backdropFilter="blur(10px)"
    >
      <Text fontSize="lg" fontWeight="bold" letterSpacing="0.04em" mb={4}>
        InsightForge
      </Text>
      <VStack align="stretch" spacing={2}>
        <NavButton label="Connections" path="/settings" icon={FiSettings} />
        <NavButton label="Sources" path="/db-viewer" icon={FiDatabase} />
        <NavButton label="Схемы" path="/schemas" icon={FiMap} />
        <NavButton label="Marts / Views" path="/builder" icon={FiLayers} disabled={!canBuilder} />
        <NavButton label="Join Builder" path="/joins" icon={FiGitBranch} disabled={!canJoins} />
        <NavButton label="Transforms" path="/transforms" icon={FiRepeat} disabled={!canTransforms} />
        <NavButton label="Review" path="/summary" icon={FiZap} disabled={!canSummary} />
      </VStack>
      <Divider my={6} borderColor="border.subtle" />
      <VStack align="stretch" spacing={2}>
        <NavButton label="Schema Changes" path="/column-mismatches" icon={FiActivity} />
        <NavButton label="Rename Suggestions" path="/column-rename-suggestions" icon={FiActivity} />
        <NavButton label="Tasks & Runs" path="/tasks" icon={FiZap} />
      </VStack>
    </Box>
  );
};

const Sidebar: React.FC<SidebarProps> = ({ isOpen, onClose, isMobile }) => {
  if (isMobile) {
    return (
      <Drawer placement="left" isOpen={isOpen} onClose={onClose} size="xs">
        <DrawerOverlay />
        <DrawerContent bg="bg.surface">
          <SidebarContent />
        </DrawerContent>
      </Drawer>
    );
  }

  return <SidebarContent />;
};

export default Sidebar;
