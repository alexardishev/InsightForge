import React from 'react';
import { Button, useColorMode } from '@chakra-ui/react';

const ThemeToggle: React.FC = () => {
  const { colorMode, toggleColorMode } = useColorMode();

  return (
    <Button onClick={toggleColorMode} variant="outline" size="sm">
      {colorMode === 'light' ? '🌙 Тёмная тема' : '☀️ Светлая тема'}
    </Button>
  );
};

export default ThemeToggle;
