import React from 'react';
import { Badge, Checkbox, HStack, Stack, Text, VStack } from '@chakra-ui/react';

interface SchemaInfo {
  name: string;
}

interface Props {
  database: string;
  schemas: SchemaInfo[];
  selectedSchemas: string[];
  onChange: (schemas: string[]) => void;
}

const SchemaSelector: React.FC<Props> = ({ database, schemas, selectedSchemas, onChange }) => {
  const toggleSchema = (schema: string) => {
    if (selectedSchemas.includes(schema)) {
      onChange(selectedSchemas.filter((item) => item !== schema));
    } else {
      onChange([...selectedSchemas, schema]);
    }
  };

  return (
    <VStack align="stretch" spacing={2} border="1px solid" borderColor="border.subtle" p={3} borderRadius="lg">
      <HStack justify="space-between" mb={1}>
        <Text fontWeight="semibold">Схемы {database}</Text>
        <Badge colorScheme="cyan">{selectedSchemas.length} выбрано</Badge>
      </HStack>
      <Stack spacing={1} maxH="200px" overflowY="auto">
        {schemas.map((schema) => (
          <HStack key={schema.name} justify="space-between">
            <Checkbox
              isChecked={selectedSchemas.includes(schema.name)}
              onChange={() => toggleSchema(schema.name)}
            >
              {schema.name}
            </Checkbox>
            {selectedSchemas.includes(schema.name) && <Badge colorScheme="green">active</Badge>}
          </HStack>
        ))}
        {schemas.length === 0 && (
          <Text color="text.muted">Нет доступных схем для этой базы данных.</Text>
        )}
      </Stack>
    </VStack>
  );
};

export default SchemaSelector;
