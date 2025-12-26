import React from 'react';
import {
  Box,
  Checkbox,
  Text,
  SimpleGrid,
  Fade,
  useColorModeValue,
  Stack,
  Badge,
  Icon,
  Button,
  Heading,
  HStack,
} from '@chakra-ui/react';
import { FaDatabase } from 'react-icons/fa';

interface Table {
  name: string;
  rows?: number;
}

interface Props {
  selectedSchemaData?: { name?: string; tables?: Table[] };
  selectedTables: string[];
  onToggleTable: (table: string) => void;
  onLoadMore?: () => void;
  hasMore?: boolean;
  isLoading?: boolean;
  dbLabel?: string;
  schemaLabel?: string;
}

const TableSelector: React.FC<Props> = ({
  selectedSchemaData,
  selectedTables,
  onToggleTable,
  onLoadMore,
  hasMore = true,
  isLoading = false,
  dbLabel,
  schemaLabel,
}) => {
  if (!selectedSchemaData) return null;

  const cardBg = useColorModeValue('gray.100', 'gray.800');
  const textColor = useColorModeValue('gray.800', 'yellow.200');
  const iconColor = useColorModeValue('teal.600', 'teal.300');
  const badgeColorScheme = useColorModeValue('green', 'teal');

  return (
    <Box w="100%" px={4} py={6}>
      <HStack mb={4} justify="space-between" flexWrap="wrap" gap={2}>
        <Heading size="md">
          {dbLabel ? `${dbLabel}.${schemaLabel}` : 'Выберите таблицы:'}
        </Heading>
        <Badge colorScheme="cyan">{selectedTables.length} выбрано</Badge>
      </HStack>

      <SimpleGrid
        columns={{ base: 1, sm: 2, md: 3, lg: 4 }}
        spacing={5}
        maxW="1200px"
        mx="auto"
      >
        {selectedSchemaData.tables?.map((table: Table) => (
          <Fade in key={table.name}>
            <Box
              p={4}
              bg={cardBg}
              borderRadius="lg"
              boxShadow="xl"
              transition="all 0.2s"
              _hover={{ transform: 'scale(1.03)', boxShadow: '2xl' }}
            >
              <Stack spacing={2}>
                <Stack direction="row" align="center" spacing={3}>
                  <Icon as={FaDatabase} boxSize={5} color={iconColor} />
                  <Checkbox
                    isChecked={selectedTables.includes(table.name)}
                    onChange={() => onToggleTable(table.name)}
                    colorScheme="teal"
                  >
                    <Text width={"50px"} fontSize="md" whiteSpace="nowrap" color={textColor}>
                      {table.name}
                    </Text>
                  </Checkbox>
                </Stack>

                {typeof table.rows === 'number' && (
                  <Badge colorScheme={badgeColorScheme} alignSelf="flex-start">
                    {table.rows} строк
                  </Badge>
                )}
              </Stack>
            </Box>
          </Fade>
        ))}
      </SimpleGrid>
      {onLoadMore && hasMore && (
        <Box textAlign="center" mt={4}>
          <Button onClick={onLoadMore} isLoading={isLoading}>
            Загрузить ещё
          </Button>
        </Box>
      )}
    </Box>
  );
};

export default TableSelector;
