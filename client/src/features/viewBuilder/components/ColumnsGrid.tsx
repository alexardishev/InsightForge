import React from 'react';
import {
  Box,
  Text,
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AccordionIcon,
  Badge,
  HStack,
  VStack,
  Divider,
  Checkbox,
  useColorModeValue
} from '@chakra-ui/react';

interface SelectedColumn {
  table: string;
  column: string;
}

interface Props {
  selectedTables: string[];
  selectedSchemaData: any;
  selectedColumns: SelectedColumn[];
  onToggleColumn: (table: string, column: string) => void;
}

const ColumnsList: React.FC<Props> = ({
  selectedTables,
  selectedSchemaData,
  selectedColumns,
  onToggleColumn
}) => {
  const panelBg = useColorModeValue('gray.50', 'gray.700');
  const panelText = useColorModeValue('gray.800', 'gray.100');
  const border = useColorModeValue('gray.200', 'gray.600');

  if (selectedTables.length === 0 || !selectedSchemaData) return null;

  return (
    <Box w="100%" px={4} py={6}>
      <Text mb={6} fontWeight="bold" textAlign="center" fontSize="xl">
        Колонки в выбранных таблицах:
      </Text>
      <Accordion allowMultiple>
        {selectedTables.map((tableName) => {
          const tableData = selectedSchemaData.tables?.find((t: any) => t.name === tableName);
          return (
            <AccordionItem key={tableName} border="1px solid" borderColor={border} borderRadius="md" mb={3}>
              <h2>
                <AccordionButton>
                  <Box flex="1" textAlign="left" fontWeight="semibold" fontSize="md">
                    {tableName}
                  </Box>
                  <AccordionIcon />
                </AccordionButton>
              </h2>
              <AccordionPanel pb={4} bg={panelBg} color={panelText}>
                <VStack align="start" spacing={4}>
                  {tableData?.columns?.map((col: any, index: number) => (
                    <Box key={index} w="100%">
                      <HStack justify="space-between" mb={1}>
                        <HStack spacing={3}>
                          <Checkbox
                            isChecked={selectedColumns.some(
                              (c) => c.table === tableName && c.column === col.name
                            )}
                            onChange={() => onToggleColumn(tableName, col.name)}
                          />
                          <Text fontWeight="medium">{col.name}</Text>
                        </HStack>
                        <HStack spacing={2} wrap="wrap">
                          <Badge colorScheme="blue">{col.type}</Badge>
                          {col.is_primary_key && <Badge colorScheme="red">PK</Badge>}
                          {col.is_fk && <Badge colorScheme="orange">FK</Badge>}
                          {col.is_unique && <Badge colorScheme="purple">UNQ</Badge>}
                        </HStack>
                      </HStack>

                      <VStack align="start" spacing={2} pl={6} mt={1}>
                        <HStack>
                          <Text fontWeight="medium" minW="90px" fontSize="sm">Тип:</Text>
                          <Text fontSize="sm">{col.type}</Text>
                        </HStack>

                        <HStack>
                          <Text fontWeight="medium" minW="90px" fontSize="sm">Nullable:</Text>
                          <Badge colorScheme={col.is_nullable ? 'yellow' : 'green'}>
                            {col.is_nullable ? 'Да' : 'Нет'}
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

                        <HStack wrap="wrap" pt={2}>
                          {col.is_primary_key && (
                            <Badge colorScheme="red" variant="outline">Первичный ключ</Badge>
                          )}
                          {col.is_fk && (
                            <Badge colorScheme="orange" variant="outline">Внешний ключ</Badge>
                          )}
                          {col.is_unique && (
                            <Badge colorScheme="purple" variant="outline">Уникальный</Badge>
                          )}
                          {!col.is_primary_key && !col.is_fk && !col.is_unique && (
                            <Text fontSize="sm" color="gray.400">Обычная колонка</Text>
                          )}
                        </HStack>

                        <Divider />
                      </VStack>
                    </Box>
                  ))}
                </VStack>
              </AccordionPanel>
            </AccordionItem>
          );
        })}
      </Accordion>
    </Box>
  );
};

export default ColumnsList;
