import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import SettingsPage from './features/settings/SettingsPage';
import { Box, Flex } from '@chakra-ui/react';
import ThemeToggle from './components/ThemeToggle';
import ViewBuilderPage from './features/viewBuilder/ViewBuilderPage';
import JoinBuilderPage from './features/viewBuilder/JoinBuilderPage';

const App: React.FC = () => {
  return (
    <Box>
      <Flex justify="flex-end" p={4}>
        <ThemeToggle />
      </Flex>

      <Routes>
        <Route path="/settings" element={<SettingsPage />} />
        <Route path='/builder' element={<ViewBuilderPage/>}></Route>
        <Route path='/joins' element={<JoinBuilderPage/>}></Route>
        <Route path="*" element={<Navigate to="/settings" replace />} />
      </Routes>
    </Box>
  );
};

export default App;
