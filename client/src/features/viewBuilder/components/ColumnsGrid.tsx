import React from 'react';
import {
  Box,
  Text,
  Button,
  Badge,
  HStack,
  VStack,
  Checkbox,
  Input,
  Stack,
  Card,
  CardBody,
  Tag,
  SimpleGrid,
} from '@chakra-ui/react';
import type { SelectedColumn } from '../viewBuilderSlice';

interface SchemaData {
  name?: string;
  tables?: any[];
}

interface SourceView {
  db: string;
  schema: string;
  selectedTables: string[];
  selectedColumns: SelectedColumn[];
  schemaData?: SchemaData;
}

interface Props {
  sources: SourceView[];
  onToggleColumn: (db: string, schema: string, table: string, column: any) => void;
  onSetTableColumns: (db: string, schema: string, table: string, columns: any[]) => void;
  onAliasChange: (db: string, schema: string, table: string, column: string, alias: string) => void;
}

const ColumnsGrid: React.FC<Props> = ({
  sources,
  onToggleColumn,
  onSetTableColumns,
  onAliasChange,
}) => {
  if (sources.every((source) => source.selectedTables.length === 0)) return null;

  return (
    <Box w="100%" px={{ base: 0, md: 2 }} py={4}>
      <VStack align="stretch" spacing={6}>
        <HStack justify="space-between" flexWrap="wrap" gap={3}>
          <Box>
            <Text mb={1} fontWeight="bold" fontSize="xl">
              Колонки витрины
            </Text>
            <Text color="text.muted" fontSize="sm">
              Чекбоксы и алиасы приведены к новому стилю. "Выбрать все" учитывает безопасные ключи.
            </Text>
          </Box>
        </HStack>

        {sources.map((source) => {
          if (source.selectedTables.length === 0 || !source.schemaData) return null;
          return (
            <Box key={`${source.db}.${source.schema}`}>
              <HStack justify="space-between" mb={3}>
                <VStack align="start" spacing={0}>
                  <Text fontWeight="semibold">
                    {source.db}.{source.schema}
                  </Text>
                  <Text color="text.muted" fontSize="sm">
                    {source.selectedTables.length} выбранных таблиц
                  </Text>
                </VStack>
                <Badge colorScheme="purple">{source.schemaData.tables?.length || 0} таблиц</Badge>
              </HStack>

              <SimpleGrid columns={{ base: 1, lg: 2 }} spacing={3}>
                {source.selectedTables.map((tableName) => {
                  const tableData = source.schemaData?.tables?.find((t: any) => t.name === tableName);
                  const allSelected = tableData?.columns?.every((col: any) =>
                    source.selectedColumns.some(
                      (c) => c.table === tableName && c.column === col.name,
                    ),
                  );

                  if (!tableData) return null;

                  return (
                    <Card
                      key={`${source.db}.${source.schema}.${tableName}`}
                      variant="surface"
                      border="1px solid"
                      borderColor="border.subtle"
                    >
                      <CardBody>
                        <HStack justify="space-between" mb={3} align="center">
                          <VStack align="start" spacing={0}>
                            <Text fontWeight="semibold">{tableName}</Text>
                            <Text color="text.muted" fontSize="sm">
                              {tableData?.columns?.length || 0} колонок
                            </Text>
                          </VStack>
                          <HStack spacing={2}>
                            <Button
                              size="sm"
                              variant="outline"
                              onClick={() =>
                                onSetTableColumns(
                                  source.db,
                                  source.schema,
                                  tableName,
                                  allSelected ? [] : tableData?.columns || [],
                                )
                              }
                            >
                              {allSelected ? 'Снять все' : 'Выбрать все'}
                            </Button>
                          </HStack>
                        </HStack>

                        <Stack spacing={2} maxH="360px" overflowY="auto">
                          {tableData?.columns?.map((col: any) => {
                            const isChecked = source.selectedColumns.some(
                              (c) => c.table === tableName && c.column === col.name,
                            );
                            const selected = source.selectedColumns.find(
                              (c) => c.table === tableName && c.column === col.name,
                            );
                            return (
                              <Box
                                key={col.name}
                                p={3}
                                border="1px solid"
                                borderColor={isChecked ? 'accent.primary' : 'border.subtle'}
                                borderRadius="lg"
                                bg="bg.elevated"
                                transition="border-color 0.2s ease, transform 0.2s ease"
                              >
                                <HStack justify="space-between" align="start" spacing={3}>
                                  <HStack align="start" spacing={3} flex="1">
                                    <Checkbox
                                      isChecked={isChecked}
                                      onChange={() => onToggleColumn(source.db, source.schema, tableName, col)}
                                      size="lg"
                                    />
                                    <VStack align="start" spacing={1} flex="1">
                                      <HStack spacing={2} wrap="wrap">
                                        <Text fontWeight="semibold">{col.name}</Text>
                                        <Tag colorScheme="cyan" variant="subtle">{col.type}</Tag>
                                        {col.is_primary_key && <Tag colorScheme="green">PK</Tag>}
                                        {col.is_fk && <Tag colorScheme="purple">FK</Tag>}
                                        {col.is_unique && <Tag colorScheme="orange">UNQ</Tag>}
                                      </HStack>
                                      <HStack spacing={2} color="text.muted" fontSize="sm">
                                        <Text>{col.is_nullable ? 'Nullable' : 'Not null'}</Text>
                                        {col.default && <Badge variant="outline">default</Badge>}
                                      </HStack>
                                      {isChecked && (
                                        <Input
                                          placeholder="Алиас колонки (опционально)"
                                          size="sm"
                                          variant="filled"
                                          value={selected?.alias || ''}
                                          onChange={(e) =>
                                            onAliasChange(
                                              source.db,
                                              source.schema,
                                              tableName,
                                              col.name,
                                              e.target.value,
                                            )
                                          }
                                        />
                                      )}
                                    </VStack>
                                  </HStack>
                                </HStack>
                              </Box>
                            );
                          })}
                        </Stack>
                      </CardBody>
                    </Card>
                  );
                })}
              </SimpleGrid>
            </Box>
          );
        })}
      </VStack>
    </Box>
  );
};

export default ColumnsGrid;
