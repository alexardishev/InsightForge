import React, { useEffect, useMemo, useState } from 'react';
import {
  Badge,
  Box,
  Card,
  CardBody,
  Flex,
  Heading,
  HStack,
  Icon,
  IconButton,
  SimpleGrid,
  Text,
  VStack,
  useColorModeValue,
  Button,
  Divider,
  Stack,
  Tooltip,
} from '@chakra-ui/react';
import { FiDatabase, FiGrid, FiKey, FiRefreshCw, FiServer } from 'react-icons/fi';
import { useSelector, useDispatch } from 'react-redux';
import type { RootState, AppDispatch } from '../../app/store';
import DatabaseSelector from '../viewBuilder/components/DatabaseSelector';
import SchemaSelector from '../viewBuilder/components/SchemaSelector';
import { useHttp } from '../../hooks/http.hook';
import { appendTables } from '../settings/settingsSlice';

interface TableRow {
  name: string;
  columns: any[];
}

const DatabaseViewerPage: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const data = useSelector((state: RootState) => state.settings.dataBaseInfo);
  const savedConnections = useSelector((state: RootState) => state.settings.savedConnections);
  const selectedConnections = useSelector(
    (state: RootState) => state.settings.selectedConnections,
  );
  const [selectedDbs, setSelectedDbs] = useState<string[]>([]);
  const [selectedSchemas, setSelectedSchemas] = useState<string[]>([]);
  const { request } = useHttp();
  const url = '/api';

  const currentDb = selectedDbs[0] || '';
  const currentSchema = selectedSchemas[0] || '';
  const selectedDatabase = data?.find((db: any) => db.name === currentDb);
  const selectedSchemaData = selectedDatabase?.schemas?.find((s: any) => s.name === currentSchema);

  const [tablesState, setTablesState] = useState<TableRow[]>(selectedSchemaData?.tables || []);
  const [page, setPage] = useState(1);
  const pageSize = 20;
  const [focusedTable, setFocusedTable] = useState<string | null>(null);

  const blueprintBg = useColorModeValue('radial-gradient(circle at 20% 20%, rgba(0, 255, 255, 0.06), transparent 35%), linear-gradient(90deg, rgba(255,255,255,0.04) 1px, transparent 1px), linear-gradient(180deg, rgba(255,255,255,0.04) 1px, transparent 1px)',
    'radial-gradient(circle at 20% 20%, rgba(0, 255, 255, 0.08), transparent 35%), linear-gradient(90deg, rgba(255,255,255,0.06) 1px, transparent 1px), linear-gradient(180deg, rgba(255,255,255,0.06) 1px, transparent 1px)');

  useEffect(() => {
    if (!currentDb && data?.length) {
      const firstDb = data[0];
      setSelectedDbs([firstDb.name]);
      if (firstDb.schemas?.length) {
        setSelectedSchemas([firstDb.schemas[0].name]);
      }
    }
  }, [currentDb, data]);

  useEffect(() => {
    if (selectedDatabase && selectedDatabase.schemas?.length && !currentSchema) {
      const fallback = selectedDatabase.schemas[0]?.name;
      if (fallback) setSelectedSchemas([fallback]);
    }
  }, [currentDb, currentSchema, selectedDatabase]);

  useEffect(() => {
    setTablesState(selectedSchemaData?.tables || []);
    setPage(1);
    setFocusedTable(null);
  }, [selectedSchemaData]);

  const loadMore = async () => {
    const nextPage = page + 1;
    try {
      const connectionStrings = selectedConnections
        .map((key) => ({ key, value: savedConnections[key] }))
        .filter((item): item is { key: string; value: string } => Boolean(item.value));

      const body = {
        connection_strings: connectionStrings.map(({ key, value }) => ({
          connection_string: {
            [key]: value,
          },
        })),
        page: nextPage,
        page_size: pageSize,
      };
      const dbInfo = await request(`${url}/get-db`, 'POST', body);
      const db = dbInfo.find((d: any) => d.name === currentDb);
      const schema = db?.schemas?.find((s: any) => s.name === currentSchema);
      const newTables = schema?.tables || [];
      if (newTables.length > 0) {
        setTablesState((prev) => [...prev, ...newTables]);
        dispatch(appendTables({ db: currentDb, schema: currentSchema, tables: newTables }));
        setPage(nextPage);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const focusedData = useMemo(
    () => tablesState.find((t) => t.name === focusedTable),
    [focusedTable, tablesState],
  );

  const renderColumnBadge = (col: any) => {
    const pk = col.is_primary_key || col.is_pk;
    const uk = col.is_unique;
    return (
      <HStack spacing={2} align="center" justify="space-between">
        <Text fontWeight="semibold">{col.name}</Text>
        <HStack spacing={1}>
          {pk && (
            <Tooltip label="Primary key" openDelay={150}>
              <Badge colorScheme="cyan">PK</Badge>
            </Tooltip>
          )}
          {uk && <Badge colorScheme="purple">UQ</Badge>}
          {col.is_nullable ? <Badge variant="outline">NULL</Badge> : <Badge colorScheme="green">NOT NULL</Badge>}
        </HStack>
      </HStack>
    );
  };

  return (
    <Box>
      <HStack justify="space-between" mb={6} flexWrap="wrap" gap={3}>
        <Heading size="lg">Интерактивная схема БД</Heading>
        <HStack spacing={2} color="text.muted" fontSize="sm">
          <Badge colorScheme="cyan">ПК — ключи</Badge>
          <Badge colorScheme="purple">UQ</Badge>
          <Badge variant="outline">NULL</Badge>
        </HStack>
      </HStack>

      <Card variant="glass" mb={4}>
        <CardBody>
          <SimpleGrid columns={{ base: 1, md: 2 }} spacing={3} alignItems="center">
            <DatabaseSelector
              data={data}
              selectedDbs={selectedDbs}
              onChange={(dbs) => setSelectedDbs(dbs.slice(-1))}
            />
            {currentDb && selectedDatabase && (
              <SchemaSelector
                database={currentDb}
                schemas={selectedDatabase.schemas || []}
                selectedSchemas={selectedSchemas}
                onChange={(schemas) => setSelectedSchemas(schemas.slice(-1))}
              />
            )}
          </SimpleGrid>
        </CardBody>
      </Card>

      {currentSchema && (
        <Stack spacing={4}>
          <Card variant="surface">
            <CardBody>
              <HStack justify="space-between" mb={3} align="center">
                <HStack>
                  <Icon as={FiDatabase} />
                  <Text fontWeight="bold">{currentDb}</Text>
                  <Badge>{currentSchema}</Badge>
                </HStack>
                <Button
                  leftIcon={<FiRefreshCw />}
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setPage(1);
                    setTablesState(selectedSchemaData?.tables || []);
                  }}
                >
                  Обновить
                </Button>
              </HStack>

              <Box
                borderRadius="xl"
                border="1px solid"
                borderColor="border.subtle"
                p={4}
                bg={blueprintBg}
                overflowX="auto"
              >
                <SimpleGrid columns={{ base: 1, md: 2, xl: 3 }} spacing={3} minW="full">
                  {tablesState.map((table) => (
                    <Card
                      key={table.name}
                      variant={focusedTable === table.name ? 'glass' : 'surface'}
                      borderColor={focusedTable === table.name ? 'accent.primary' : 'border.subtle'}
                      cursor="pointer"
                      onClick={() => setFocusedTable(table.name)}
                    >
                      <CardBody>
                        <HStack justify="space-between" mb={2}>
                          <HStack spacing={2}>
                            <Icon as={FiGrid} />
                            <Text fontWeight="bold">{table.name}</Text>
                          </HStack>
                          <Badge colorScheme="cyan">{table.columns.length} полей</Badge>
                        </HStack>
                        <VStack align="stretch" spacing={1} maxH="200px" overflowY="auto">
                          {table.columns.slice(0, 8).map((col) => (
                            <Box
                              key={col.name}
                              border="1px solid"
                              borderColor="border.subtle"
                              borderRadius="md"
                              px={2}
                              py={1}
                              bg="bg.elevated"
                            >
                              {renderColumnBadge(col)}
                              <Text fontSize="xs" color="text.muted">
                                {col.type}
                              </Text>
                            </Box>
                          ))}
                          {table.columns.length > 8 && (
                            <Text fontSize="xs" color="text.muted">
                              + ещё {table.columns.length - 8} колонок
                            </Text>
                          )}
                        </VStack>
                      </CardBody>
                    </Card>
                  ))}
                </SimpleGrid>
              </Box>

              <Flex justify="center" mt={4}>
                <Button variant="ghost" onClick={loadMore} leftIcon={<FiRefreshCw />}>
                  Загрузить ещё
                </Button>
              </Flex>
            </CardBody>
          </Card>

          {focusedData && (
            <Card variant="glass">
              <CardBody>
                <HStack justify="space-between" mb={2}>
                  <HStack spacing={2}>
                    <Icon as={FiServer} />
                    <Text fontWeight="bold">{focusedData.name}</Text>
                    <Badge colorScheme="cyan">{focusedData.columns.length} колонок</Badge>
                  </HStack>
                  <IconButton
                    aria-label="close"
                    icon={<FiRefreshCw />}
                    variant="ghost"
                    onClick={() => setFocusedTable(null)}
                  />
                </HStack>
                <Divider mb={3} />
                <SimpleGrid columns={{ base: 1, md: 2, xl: 3 }} spacing={3}>
                  {focusedData.columns.map((col) => (
                    <Card key={col.name} variant="surface">
                      <CardBody>
                        <HStack justify="space-between" mb={1}>
                          <HStack spacing={2}>
                            {(col.is_primary_key || col.is_pk) && <Icon as={FiKey} color="accent.primary" />}
                            <Text fontWeight="semibold">{col.name}</Text>
                          </HStack>
                          <Badge>{col.type}</Badge>
                        </HStack>
                        <Text fontSize="sm" color="text.muted">
                          {col.is_nullable ? 'Nullable' : 'Not null'} {col.is_unique && '• Unique'}
                        </Text>
                      </CardBody>
                    </Card>
                  ))}
                </SimpleGrid>
              </CardBody>
            </Card>
          )}
        </Stack>
      )}
    </Box>
  );
};

export default DatabaseViewerPage;
