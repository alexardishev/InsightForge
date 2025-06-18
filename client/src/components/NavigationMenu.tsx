import React from 'react';
import { HStack, Button } from '@chakra-ui/react';
import { NavLink } from 'react-router-dom';
import { useSelector } from 'react-redux';
import type { RootState } from '../app/store';

const NavigationMenu: React.FC = () => {
  const settings = useSelector((state: RootState) => state.settings);
  const builder = useSelector((state: RootState) => state.viewBuilder);

  const canBuilder = !!settings.dataBaseInfo;
  const canJoins = builder.selectedColumns.length > 0;
  const canTransforms = canJoins;
  const canSummary = canTransforms;

  const links = [
    { label: 'Подключение', path: '/settings', enabled: true },
    { label: 'Таблицы', path: '/builder', enabled: canBuilder },
    { label: 'Джоины', path: '/joins', enabled: canJoins },
    { label: 'Трансформации', path: '/transforms', enabled: canTransforms },
    { label: 'Итог', path: '/summary', enabled: canSummary },
  ];

  return (
    <HStack spacing={4} justify="center" mb={4}>
      {links.map((link) => (
        <Button
          as={NavLink}
          key={link.path}
          to={link.path}
          variant="outline"
          colorScheme="teal"
          isDisabled={!link.enabled}
        >
          {link.label}
        </Button>
      ))}
    </HStack>
  );
};

export default NavigationMenu;
