import React from 'react';
import {
  IconButton,
  useDisclosure,
  Drawer,
  DrawerOverlay,
  DrawerContent,
  DrawerHeader,
  DrawerBody,
  Button,
  VStack,
} from '@chakra-ui/react';
import { HamburgerIcon } from '@chakra-ui/icons';
import { NavLink } from 'react-router-dom';
import { useSelector } from 'react-redux';
import type { RootState } from '../app/store';

const NavigationMenu: React.FC = () => {
  const { isOpen, onOpen, onClose } = useDisclosure();
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
    { label: 'Переименования колонок', path: '/column-rename-suggestions', enabled: true },
    { label: 'Таблицы', path: '/builder', enabled: canBuilder },
    { label: 'Джоины', path: '/joins', enabled: canJoins },
    { label: 'Трансформации', path: '/transforms', enabled: canTransforms },
    { label: 'Итог', path: '/summary', enabled: canSummary },
  ];

  return (
    <>
      <IconButton
        icon={<HamburgerIcon />}
        variant="outline"
        aria-label="Navigation menu"
        mb={4}
        onClick={onOpen}
      />
      <Drawer placement="left" onClose={onClose} isOpen={isOpen}>
        <DrawerOverlay />
        <DrawerContent>
          <DrawerHeader borderBottomWidth="1px">Меню</DrawerHeader>
          <DrawerBody>
            <VStack align="start" spacing={3} mt={2}>
              {links.map((link) => (
                <Button
                  as={NavLink}
                  key={link.path}
                  to={link.path}
                  onClick={onClose}
                  isDisabled={!link.enabled}
                  variant="ghost"
                  width="100%"
                  _activeLink={{ bg: 'blue.500', color: 'white' }}
                >
                  {link.label}
                </Button>
              ))}
            </VStack>
          </DrawerBody>
        </DrawerContent>
      </Drawer>
    </>
  );
};

export default NavigationMenu;
