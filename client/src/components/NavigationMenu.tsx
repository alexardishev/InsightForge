import React from 'react';
import {
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  IconButton,
} from '@chakra-ui/react';
import { HamburgerIcon } from '@chakra-ui/icons';
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
    { label: 'Просмотр БД', path: '/db-viewer', enabled: true },
    { label: 'Подключение', path: '/settings', enabled: true },
    { label: 'Задачи', path: '/tasks', enabled: true },
    { label: 'Таблицы', path: '/builder', enabled: canBuilder },
    { label: 'Джоины', path: '/joins', enabled: canJoins },
    { label: 'Трансформации', path: '/transforms', enabled: canTransforms },
    { label: 'Итог', path: '/summary', enabled: canSummary },
  ];

  return (
    <Menu>
      <MenuButton
        as={IconButton}
        icon={<HamburgerIcon />}
        variant="outline"
        aria-label="Navigation menu"
        mb={4}
      />
      <MenuList>
        {links.map((link) => (
          <MenuItem
            as={NavLink}
            key={link.path}
            to={link.path}
            isDisabled={!link.enabled}
          >
            {link.label}
          </MenuItem>
        ))}
      </MenuList>
    </Menu>
  );
};

export default NavigationMenu;
