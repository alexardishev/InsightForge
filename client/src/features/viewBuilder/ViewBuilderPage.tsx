import React, { useState } from 'react';
import {
  Box,
  Heading,
  Select,
  VStack,
  Checkbox,
  Text,
  SimpleGrid,
} from '@chakra-ui/react';

const mockData: {
  databases: string[];
  schemas: Record<string, string[]>;
  tables: Record<string, string[]>;
  columns: Record<string, string[]>;
} = {
  databases: ['main_db', 'test_db'],
  schemas: {
    main_db: ['public', 'analytics'],
    test_db: ['default'],
  },
  tables: {
    public: ['users', 'orders'],
    analytics: ['events', 'clicks'],
    default: ['test_table'],
  },
  columns: {
    users: ['id', 'name', 'email'],
    orders: ['id', 'amount', 'date'],
    events: ['id', 'type', 'timestamp'],
    clicks: ['id', 'element', 'timestamp'],
    test_table: ['id', 'value'],
  },
};

const ViewBuilderPage: React.FC = () => {
  const [selectedDb, setSelectedDb] = useState('');
  const [selectedSchema, setSelectedSchema] = useState('');
  const [selectedTables, setSelectedTables] = useState<string[]>([]);

  const handleToggleTable = (table: string) => {
    setSelectedTables((prev) =>
      prev.includes(table) ? prev.filter((t) => t !== table) : [...prev, table]
    );
  };

  return (
    <Box p={8}>
      <Heading mb={6}>Конструктор витрины</Heading>
      <VStack align="stretch" spacing={4} maxW="md">
        <Select
          placeholder="Выберите базу данных"
          value={selectedDb}
          onChange={(e) => {
            setSelectedDb(e.target.value);
            setSelectedSchema('');
            setSelectedTables([]);
          }}
        >
          {mockData.databases.map((db) => (
            <option key={db} value={db}>
              {db}
            </option>
          ))}
        </Select>

        {selectedDb && (
          <Select
            placeholder="Выберите схему"
            value={selectedSchema}
            onChange={(e) => {
              setSelectedSchema(e.target.value);
              setSelectedTables([]);
            }}
          >
            {(mockData.schemas as Record<string, string[]>)[selectedDb]?.map((schema) => (
              <option key={schema} value={schema}>
                {schema}
              </option>
            ))}
          </Select>
        )}

        {selectedSchema && (
          <>
            <Text>Выберите таблицы:</Text>
            <VStack align="start">
              {(mockData.tables as Record<string, string[]>)[selectedSchema]?.map((table) => (
                <Checkbox
                  key={table}
                  isChecked={selectedTables.includes(table)}
                  onChange={() => handleToggleTable(table)}
                >
                  {table}
                </Checkbox>
              ))}
            </VStack>
          </>
        )}

        {selectedTables.length > 0 && (
          <>
            <Text pt={4}>Колонки в выбранных таблицах:</Text>
            <SimpleGrid columns={2} spacing={4}>
              {selectedTables.map((table) => (
                <Box key={table} p={2} borderWidth="1px" borderRadius="md">
                  <Text fontWeight="bold">{table}</Text>
                  <VStack align="start" mt={2}>
                    {mockData.columns[table]?.map((col) => (
                      <Text key={col} fontSize="sm">
                        • {col}
                      </Text>
                    ))}
                  </VStack>
                </Box>
              ))}
            </SimpleGrid>
          </>
        )}
      </VStack>
    </Box>
  );
};

export default ViewBuilderPage;
