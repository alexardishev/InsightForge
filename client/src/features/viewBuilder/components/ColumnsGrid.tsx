import React from 'react';
import {
  Box,
  Text,
  SimpleGrid,
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

const ColumnsGrid: React.FC<Props> = ({
  selectedTables,
  selectedSchemaData,
  selectedColumns,
  onToggleColumn
}) => {
  const panelBg = useColorModeValue('gray.50', 'gray.700');
  const panelText = useColorModeValue('gray.800', 'gray.100');
  const cardBg = useColorModeValue('white', 'gray.800');
  const cardBorder = useColorModeValue('gray.200', 'gray.600');

  if (selectedTables.length === 0 || !selectedSchemaData) return null;

  return (
    <Box w="100%">
      <Text pt={4} mb={6} fontWeight="medium" textAlign="center" fontSize="lg">
        Колонки в выбранных таблицах:
      </Text>
      <SimpleGrid
        columns={{ base: 1, lg: 2, xl: 3 }}
        spacing={6}
        alignItems="start"
        gridAutoRows="max-content"
      >
        {selectedTables.map((tableName) => {
          const tableData = selectedSchemaData.tables?.find((t: any) => t.name === tableName);
          return (
            <Box
              key={tableName}
              p={4}
              borderWidth="1px"
              borderColor={cardBorder}
              borderRadius="lg"
              bg={cardBg}
              shadow="md"
              _hover={{ shadow: 'lg' }}
              transition="all 0.2s"
            >
              <Text fontWeight="bold" fontSize="lg" mb={4} textAlign="center" color="blue.400">
                {tableName}
              </Text>
              <Accordion allowMultiple>
                {tableData?.columns?.map((col: any, index: number) => (
                  <AccordionItem key={col.name || index} border="none">
                    <AccordionButton _hover={{ bg: useColorModeValue('gray.100', 'gray.600') }} borderRadius="md" mb={1}>
                      <Checkbox
                        mr={4}
                        isChecked={selectedColumns.some((c) => c.table === tableName && c.column === col.name)}
                        onChange={() => onToggleColumn(tableName, col.name)}
                      />
                      <Box flex="1" textAlign="left">
                        <VStack align="start" spacing={1}>
                          <Text fontWeight="medium">{col.name}</Text>
                          <HStack wrap="wrap">
                            <Badge colorScheme="blue" variant="solid" size="sm">
                              {col.type}
                            </Badge>
                            {col.is_primary_key && (
                              <Badge colorScheme="red" variant="solid" size="sm">
                                PK
                              </Badge>
                            )}
                            {col.is_fk && (
                              <Badge colorScheme="orange" variant="solid" size="sm">
                                FK
                              </Badge>
                            )}
                            {col.is_unique && (
                              <Badge colorScheme="purple" variant="solid" size="sm">
                                UNQ
                              </Badge>
                            )}
                          </HStack>
                        </VStack>
                      </Box>
                      <AccordionIcon />
                    </AccordionButton>
                    <AccordionPanel pb={4} bg={panelBg} color={panelText} borderRadius="md" mt={1}>
                      <VStack align="start" spacing={3}>
                        <HStack>
                          <Text fontWeight="medium" minW="80px" fontSize="sm">
                            Тип:
                          </Text>
                          <Text fontSize="sm">{col.type}</Text>
                        </HStack>
                        <HStack>
                          <Text fontWeight="medium" minW="80px" fontSize="sm">
                            Nullable:
                          </Text>
                          <Badge colorScheme={col.is_nullable ? 'yellow' : 'green'} size="sm">
                            {col.is_nullable ? 'Да' : 'Нет'}
                          </Badge>
                        </HStack>
                        {col.default && (
                          <VStack align="start" spacing={1}>
                            <Text fontWeight="medium" fontSize="sm">
                              По умолчанию:
                            </Text>
                            <Text fontSize="xs" p={2} borderRadius="md" wordBreak="break-word" bg="gray.600">
                              {col.default}
                            </Text>
                          </VStack>
                        )}
                        {col.description && (
                          <VStack align="start" spacing={1}>
                            <Text fontWeight="medium" fontSize="sm">
                              Описание:
                            </Text>
                            <Text fontSize="sm">{col.description}</Text>
                          </VStack>
                        )}
                        <Divider />
                        <VStack align="start" spacing={2}>
                          <Text fontWeight="medium" fontSize="sm">
                            Свойства:
                          </Text>
                          <HStack wrap="wrap">
                            {col.is_pk && (
                              <Badge colorScheme="red" variant="outline" size="sm">
                                Первичный ключ
                              </Badge>
                            )}
                            {col.is_fk && (
                              <Badge colorScheme="orange" variant="outline" size="sm">
                                Внешний ключ
                              </Badge>
                            )}
                            {col.is_unique && (
                              <Badge colorScheme="purple" variant="outline" size="sm">
                                Уникальный
                              </Badge>
                            )}
                            {!col.is_pk && !col.is_fk && !col.is_unique && (
                              <Text fontSize="sm" color="gray.400">
                                Обычная колонка
                              </Text>
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
  );
};

export default ColumnsGrid;
