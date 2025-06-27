import React from 'react';
import { Box, Checkbox, Text, SimpleGrid } from '@chakra-ui/react';

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
      <SimpleGrid columns={{ base: 1, sm: 2, md: 3 }} spacing={3}>
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
      </SimpleGrid>
    </Box>
  );
};

export default TableSelector;
