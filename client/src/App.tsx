import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import SettingsPage from './features/settings/SettingsPage';
import { Box, Flex } from '@chakra-ui/react';
import ThemeToggle from './components/ThemeToggle';
import NavigationMenu from './components/NavigationMenu';
import ViewBuilderPage from './features/viewBuilder/ViewBuilderPage';
import JoinBuilderPage from './features/viewBuilder/JoinBuilderPage';
import TransformBuilderPage from './features/viewBuilder/TransformBuilderPage';
import SummaryPage from './features/summary/SummaryPage';
import DatabaseViewerPage from './features/dbViewer/DatabaseViewerPage';
import TasksPage from './features/tasks/TasksPage';
import ColumnRenameSuggestionsPage from './features/columnRenameSuggestions/ColumnRenameSuggestionsPage';
import ColumnMismatchListPage from './features/columnMismatches/ColumnMismatchListPage';
import ColumnMismatchDetailsPage from './features/columnMismatches/ColumnMismatchDetailsPage';

const App: React.FC = () => {
  return (
    <Box>
      <Flex justify="flex-end" p={4}>
        <ThemeToggle />
      </Flex>
      <NavigationMenu />

      <Routes>
        <Route path="/db-viewer" element={<DatabaseViewerPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/tasks" element={<TasksPage />} />
        <Route path="/column-rename-suggestions" element={<ColumnRenameSuggestionsPage />} />
        <Route path="/column-mismatches" element={<ColumnMismatchListPage />} />
        <Route path="/column-mismatches/:id" element={<ColumnMismatchDetailsPage />} />
        <Route path='/builder' element={<ViewBuilderPage/>}></Route>
        <Route path='/joins' element={<JoinBuilderPage/>}></Route>
        <Route path='/transforms' element={<TransformBuilderPage/>}></Route>
        <Route path='/summary' element={<SummaryPage/>}></Route>
        <Route path="*" element={<Navigate to="/settings" replace />} />
      </Routes>
    </Box>
  );
};

export default App;
