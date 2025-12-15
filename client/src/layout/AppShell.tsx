import React, { useState } from 'react';
import { Box, Flex, useBreakpointValue } from '@chakra-ui/react';
import Sidebar from './Sidebar';
import Topbar from './Topbar';

interface AppShellProps {
  children: React.ReactNode;
}

const AppShell: React.FC<AppShellProps> = ({ children }) => {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const isMobile = useBreakpointValue({ base: true, lg: false }) ?? false;

  return (
    <Flex minH="100vh" bg="bg.canvas" color="text.primary">
      {!isMobile && <Sidebar isOpen onClose={() => setIsSidebarOpen(false)} />}
      {isMobile && (
        <Sidebar
          isOpen={isSidebarOpen}
          onClose={() => setIsSidebarOpen(false)}
          isMobile
        />
      )}
      <Flex direction="column" flex="1" minW={0}>
        <Topbar onOpenMenu={() => setIsSidebarOpen(true)} />
        <Box as="main" px={{ base: 4, lg: 6, xl: 10 }} py={6}>
          {children}
        </Box>
      </Flex>
    </Flex>
  );
};

export default AppShell;
