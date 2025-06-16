import React, { useState } from 'react';
import {
  Box,
  Heading,
  Select,
  VStack,
  Checkbox,
  Text,
  SimpleGrid,
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AccordionIcon,
  Badge,
  HStack,
  Divider,
  Button,
  useColorModeValue,
} from '@chakra-ui/react';
import { useSelector } from 'react-redux';
import type { RootState } from '../../app/store';

interface Column {
  name: string;
  type?: string;
  is_nullable?: boolean;
  is_primary_key?: boolean;
  is_pk?: boolean;
  is_fk?: boolean;
  default?: string;
  is_unique?: boolean;
  description?: string;
}

interface Table {
  name: string;
  columns: Column[];
}

interface Schema {
  name: string;
  tables: Table[];
}

interface Source {
  name: string;
  schemas: Schema[];
}

interface View {
  view_name: string;
  sources: Source[];
  joins: any[];
}

interface SelectedColumn {
  table: string;
  column: string;
}

const ViewBuilderPage: React.FC = () => {
  const [selectedDb, setSelectedDb] = useState('');
  const [selectedSchema, setSelectedSchema] = useState('');
  const [selectedTables, setSelectedTables] = useState<string[]>([]);
  const [selectedColumns, setSelectedColumns] = useState<SelectedColumn[]>([]);
  const data = useSelector((state: RootState) => state.settings.dataBaseInfo);

  const panelBg = useColorModeValue('gray.50', 'gray.700');
  const panelText = useColorModeValue('gray.800', 'gray.100');
  const cardBg = useColorModeValue('white', 'gray.800');
  const cardBorder = useColorModeValue('gray.200', 'gray.600');

  const handleToggleTable = (table: string) => {
    setSelectedTables((prev) =>
      prev.includes(table) ? prev.filter((t) => t !== table) : [...prev, table]
    );
    setSelectedColumns((prev) => prev.filter((c) => c.table !== table));
  };

  const handleToggleColumn = (table: string, column: string) => {
    const exists = selectedColumns.find((c) => c.table === table && c.column === column);
    if (exists) {
      setSelectedColumns((prev) => prev.filter((c) => !(c.table === table && c.column === column)));
    } else {
      setSelectedColumns((prev) => [...prev, { table, column }]);
    }
  };

  const selectedDatabase = data?.find((db: any) => db.name === selectedDb);
  const selectedSchemaData = selectedDatabase?.schemas?.find((schema: any) => schema.name === selectedSchema);

  const handleBuildView = () => {
    const source: Source = {
      name: selectedDb,
      schemas: [
        {
          name: selectedSchema,
          tables: selectedTables.map((tableName) => {
            const tableData = selectedSchemaData.tables.find((t: any) => t.name === tableName);
            return {
              name: tableName,
              columns: tableData.columns
                .filter((col: any) =>
                  selectedColumns.some((c) => c.table === tableName && c.column === col.name)
                )
                .map((col: any) => ({
                  name: col.name,
                  type: col.type,
                  is_nullable: col.is_nullable,
                  is_primary_key: col.is_primary_key || col.is_pk,
                  is_fk: col.is_fk,
                  default: col.default,
                  is_unq: col.is_unique,
                })),
            };
          }),
        },
      ],
    };

    const view: View = {
      view_name: 'MyView',
      sources: [source],
      joins: [],
    };

    console.log('Собранная витрина:', view);
  };

  return (
    <Box p={8} maxW="1200px" mx="auto">
      <Heading mb={8} textAlign="center">Конструктор витрины</Heading>
      <VStack align="stretch" spacing={6}>
        <Box maxW="md" mx="auto">
          <Select
            placeholder={data?.length > 0 ? "Выберите базу данных" : "Нет доступных баз данных"}
            value={selectedDb}
            onChange={(e) => {
              setSelectedDb(e.target.value);
              setSelectedSchema('');
              setSelectedTables([]);
              setSelectedColumns([]);
            }}
          >
            {data?.map((db: any, index: number) => (
              <option key={index} value={db?.name}>
                {db?.name}
              </option>
            ))}
          </Select>
        </Box>

        {selectedDb && selectedDatabase && (
          <Box maxW="md" mx="auto">
            <Select
              placeholder="Выберите схему"
              value={selectedSchema}
              onChange={(e) => {
                setSelectedSchema(e.target.value);
                setSelectedTables([]);
                setSelectedColumns([]);
              }}
            >
              {selectedDatabase.schemas?.map((schema: any, index: number) => (
                <option key={index} value={schema?.name}>
                  {schema?.name}
                </option>
              ))}
            </Select>
          </Box>
        )}

        {selectedSchema && selectedSchemaData && (
          <Box maxW="md" mx="auto">
            <Text mb={4} fontWeight="medium" textAlign="center">Выберите таблицы:</Text>
            <VStack align="start" spacing={3}>
              {selectedSchemaData.tables?.map((table: any) => (
                <Checkbox
                  key={table.name}
                  isChecked={selectedTables.includes(table.name)}
                  onChange={() => handleToggleTable(table.name)}
                  size="lg"
                >
                  <Text fontSize="md">{table.name}</Text>
                </Checkbox>
              ))}
            </VStack>
          </Box>
        )}

        {selectedTables.length > 0 && selectedSchemaData && (
          <Box w="100%">
            <Text pt={4} mb={6} fontWeight="medium" textAlign="center" fontSize="lg">
              Колонки в выбранных таблицах:
            </Text>
            <SimpleGrid columns={{ base: 1, lg: 2, xl: 3 }} spacing={6}>
              {selectedTables.map((tableName) => {
                const tableData = selectedSchemaData.tables?.find((table: any) => table.name === tableName);
                return (
                  <Box
                    key={tableName}
                    p={4}
                    borderWidth="1px"
                    borderColor={cardBorder}
                    borderRadius="lg"
                    bg={cardBg}
                    shadow="md"
                    _hover={{ shadow: "lg" }}
                    transition="all 0.2s"
                  >
                    <Text fontWeight="bold" fontSize="lg" mb={4} textAlign="center" color="blue.400">
                      {tableName}
                    </Text>
                    <Accordion allowMultiple>
                      {tableData?.columns?.map((col: any, index: number) => (
                        <AccordionItem key={col.name || index} border="none">
                          <AccordionButton
                            _hover={{ bg: useColorModeValue("gray.100", "gray.600") }}
                            borderRadius="md"
                            mb={1}
                          >
                            <Checkbox
                              mr={4}
                              isChecked={selectedColumns.some((c) => c.table === tableName && c.column === col.name)}
                              onChange={() => handleToggleColumn(tableName, col.name)}
                            />
                            <Box flex="1" textAlign="left">
                              <VStack align="start" spacing={1}>
                                <Text fontWeight="medium">{col.name}</Text>
                                <HStack wrap="wrap">
                                  <Badge colorScheme="blue" variant="solid" size="sm">{col.type}</Badge>
                                  {col.is_primary_key && (
                                    <Badge colorScheme="red" variant="solid" size="sm">PK</Badge>
                                  )}
                                  {col.is_fk && (
                                    <Badge colorScheme="orange" variant="solid" size="sm">FK</Badge>
                                  )}
                                  {col.is_unique && (
                                    <Badge colorScheme="purple" variant="solid" size="sm">UNQ</Badge>
                                  )}
                                </HStack>
                              </VStack>
                            </Box>
                            <AccordionIcon />
                          </AccordionButton>
                          <AccordionPanel pb={4} bg={panelBg} color={panelText} borderRadius="md" mt={1}>
                            <VStack align="start" spacing={3}>
                              <HStack>
                                <Text fontWeight="medium" minW="80px" fontSize="sm">Тип:</Text>
                                <Text fontSize="sm">{col.type}</Text>
                              </HStack>
                              <HStack>
                                <Text fontWeight="medium" minW="80px" fontSize="sm">Nullable:</Text>
                                <Badge colorScheme={col.is_nullable ? "yellow" : "green"} size="sm">
                                  {col.is_nullable ? "Да" : "Нет"}
                                </Badge>
                              </HStack>
                              {col.default && (
                                <VStack align="start" spacing={1}>
                                  <Text fontWeight="medium" fontSize="sm">По умолчанию:</Text>
                                  <Text fontSize="xs" p={2} borderRadius="md" wordBreak="break-word" bg="gray.600">
                                    {col.default}
                                  </Text>
                                </VStack>
                              )}
                              {col.description && (
                                <VStack align="start" spacing={1}>
                                  <Text fontWeight="medium" fontSize="sm">Описание:</Text>
                                  <Text fontSize="sm">{col.description}</Text>
                                </VStack>
                              )}
                              <Divider />
                              <VStack align="start" spacing={2}>
                                <Text fontWeight="medium" fontSize="sm">Свойства:</Text>
                                <HStack wrap="wrap">
                                  {col.is_pk && (
                                    <Badge colorScheme="red" variant="outline" size="sm">Первичный ключ</Badge>
                                  )}
                                  {col.is_fk && (
                                    <Badge colorScheme="orange" variant="outline" size="sm">Внешний ключ</Badge>
                                  )}
                                  {col.is_unique && (
                                    <Badge colorScheme="purple" variant="outline" size="sm">Уникальный</Badge>
                                  )}
                                  {!col.is_pk && !col.is_fk && !col.is_unique && (
                                    <Text fontSize="sm" color="gray.400">Обычная колонка</Text>
                                  )}
                                </HStack>
                              </VStack>
                            </VStack>
                          </AccordionPanel>
                        </AccordionItem>
                      ))}
                    </Accordion>
                  </Box>
                );
              })}
            </SimpleGrid>
          </Box>
        )}

        {selectedColumns.length > 0 && (
          <Box textAlign="center">
            <Button onClick={handleBuildView} colorScheme="blue" size="lg">
              Собрать витрину
            </Button>
          </Box>
        )}
      </VStack>
    </Box>
  );
};

export default ViewBuilderPage;
