import React from 'react';
import { Box, Checkbox, Text, VStack } from '@chakra-ui/react';

interface Props {
  selectedSchemaData: any;
  selectedTables: string[];
  onToggleTable: (table: string) => void;
}

const TableSelector: React.FC<Props> = ({ selectedSchemaData, selectedTables, onToggleTable }) => {
  if (!selectedSchemaData) return null;

  return (
    <Box maxW="md" mx="auto">
      <Text mb={4} fontWeight="medium" textAlign="center">
        Выберите таблицы:
      </Text>
      <VStack align="start" spacing={3}>
        {selectedSchemaData.tables?.map((table: any) => (
          <Checkbox
            key={table.name}
            isChecked={selectedTables.includes(table.name)}
            onChange={() => onToggleTable(table.name)}
            size="lg"
          >
            <Text fontSize="md">{table.name}</Text>
          </Checkbox>
        ))}
      </VStack>
    </Box>
  );
};

export default TableSelector;
