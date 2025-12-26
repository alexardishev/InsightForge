import React from 'react';
import {
  Badge,
  Box,
  Button,
  Checkbox,
  HStack,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Text,
  VStack,
} from '@chakra-ui/react';
import { ChevronDownIcon } from '@chakra-ui/icons';

interface DatabaseInfo {
  name: string;
}

interface Props {
  data: DatabaseInfo[];
  selectedDbs: string[];
  onChange: (dbs: string[]) => void;
}

const DatabaseSelector: React.FC<Props> = ({ data, selectedDbs, onChange }) => {
  const toggleDb = (db: string) => {
    if (selectedDbs.includes(db)) {
      onChange(selectedDbs.filter((item) => item !== db));
    } else {
      onChange([...selectedDbs, db]);
    }
  };

  const buttonLabel = selectedDbs.length
    ? `Выбрано БД: ${selectedDbs.length}`
    : 'Выберите базы данных';

  return (
    <Menu closeOnSelect={false}>
      <MenuButton as={Button} rightIcon={<ChevronDownIcon />} variant="outline">
        {buttonLabel}
      </MenuButton>
      <MenuList maxH="320px" overflowY="auto" minW="280px">
        {data?.length === 0 && (
          <MenuItem isDisabled>Нет доступных подключений</MenuItem>
        )}
        {data?.map((db) => (
          <MenuItem key={db.name} closeOnSelect={false}>
            <HStack justify="space-between" w="100%">
              <Checkbox
                isChecked={selectedDbs.includes(db.name)}
                onChange={() => toggleDb(db.name)}
              >
                {db.name}
              </Checkbox>
              {selectedDbs.includes(db.name) && <Badge colorScheme="green">выбрано</Badge>}
            </HStack>
          </MenuItem>
        ))}
        {selectedDbs.length > 0 && (
          <Box px={3} py={2} borderTop="1px solid" borderColor="border.subtle">
            <VStack align="start" spacing={1}>
              <Text fontSize="sm" color="text.muted">
                Активные подключения
              </Text>
              <HStack spacing={2} flexWrap="wrap">
                {selectedDbs.map((db) => (
                  <Badge key={db} colorScheme="cyan">
                    {db}
                  </Badge>
                ))}
              </HStack>
            </VStack>
          </Box>
        )}
      </MenuList>
    </Menu>
  );
};

export default DatabaseSelector;
