import React, { useState} from 'react';
import {
  Box,
  Heading,
  Select,
  VStack,
  Checkbox,
  Text,
  SimpleGrid,
} from '@chakra-ui/react';
import { useSelector } from 'react-redux';
import type { RootState } from '../../app/store';

const ViewBuilderPage: React.FC = () => {
  const [selectedDb, setSelectedDb] = useState('');
  const [selectedSchema, setSelectedSchema] = useState('');
  const [selectedTables, setSelectedTables] = useState<string[]>([]);
const data = useSelector((state: RootState) => state.settings.dataBaseInfo)
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
          {data?.databases.map((db: any) => (
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
            {(data.schemas as Record<string, string[]>)[selectedDb]?.map((schema) => (
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
              {(data.tables as Record<string, string[]>)[selectedSchema]?.map((table) => (
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
                    {data.columns[table]?.map((col: any) => (
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
