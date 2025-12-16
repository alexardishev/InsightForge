import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Badge,
  Box,
  Button,
  Card,
  CardBody,
  Flex,
  Heading,
  Input,
  Select,
  Spinner,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useToast,
  HStack,
  Tag,
  Stack,
  Icon,
} from '@chakra-ui/react';
import { useNavigate } from 'react-router-dom';
import { useHttp } from '../../hooks/http.hook';
import { FiAlertTriangle, FiCheckCircle, FiDatabase, FiRefreshCw } from 'react-icons/fi';

export interface ColumnMismatchGroup {
  id: number;
  schema_id: number;
  database_name: string;
  schema_name: string;
  table_name: string;
  status: string;
  created_at: string;
  resolved_at?: string | null;
}

interface ColumnMismatchGroupsResponse {
  items: ColumnMismatchGroup[];
  limit: number;
  offset: number;
}

const ColumnMismatchListPage: React.FC = () => {
  const { request, loading } = useHttp();
  const navigate = useNavigate();
  const toast = useToast();

  const [groups, setGroups] = useState<ColumnMismatchGroup[]>([]);
  const [limit, setLimit] = useState(50);
  const [offset, setOffset] = useState(0);
  const [status, setStatus] = useState<'open' | 'resolved' | 'all'>('open');
  const [database, setDatabase] = useState('');
  const [schema, setSchema] = useState('');
  const [table, setTable] = useState('');
  const [error, setError] = useState<string | null>(null);

  const page = useMemo(() => Math.floor(offset / limit) + 1, [offset, limit]);

  const fetchGroups = useCallback(async () => {
    setError(null);
    const params = new URLSearchParams({
      limit: limit.toString(),
      offset: offset.toString(),
    });

    if (status !== 'all') {
      params.set('status', status);
    }
    if (database) params.set('database', database);
    if (schema) params.set('schema', schema);
    if (table) params.set('table', table);

    try {
      const data = await request<ColumnMismatchGroupsResponse>(`/api/column-mismatch-groups?${params.toString()}`);
      setGroups(data.items || []);
    } catch (e) {
      console.error(e);
      setError('Не удалось загрузить группы рассинхронов');
      toast({ title: 'Ошибка загрузки', status: 'error', duration: 3000, isClosable: true });
    }
  }, [database, limit, offset, request, schema, status, table, toast]);

  useEffect(() => {
    fetchGroups();
  }, [fetchGroups]);

  const handleResetFilters = () => {
    setStatus('open');
    setDatabase('');
    setSchema('');
    setTable('');
    setOffset(0);
  };

  const formatDate = (value?: string | null) => {
    if (!value) return '-';
    return new Date(value).toLocaleString();
  };

  return (
    <Box>
      <Flex justify="space-between" align="center" mb={6} wrap="wrap" gap={4}>
        <Heading size="lg">Контроль рассинхронов схемы</Heading>
        <HStack spacing={2} color="text.muted">
          <Tag colorScheme="yellow" variant="subtle" display="flex" alignItems="center" gap={2}>
            <Icon as={FiAlertTriangle} /> Открытые
          </Tag>
          <Tag colorScheme="green" variant="subtle" display="flex" alignItems="center" gap={2}>
            <Icon as={FiCheckCircle} /> Закрытые
          </Tag>
        </HStack>
      </Flex>

      <Card variant="glass" mb={6}>
        <CardBody>
          <Stack direction={{ base: 'column', md: 'row' }} spacing={4} align="flex-end" wrap="wrap">
            <Box flex="1" minW="160px">
              <Text fontWeight="medium" mb={1}>Статус</Text>
              <Select value={status} onChange={(e) => setStatus(e.target.value as typeof status)}>
                <option value="open">Открытые</option>
                <option value="resolved">Закрытые</option>
                <option value="all">Все</option>
              </Select>
            </Box>
            <Box flex="1" minW="160px">
              <Text fontWeight="medium" mb={1}>Database</Text>
              <Input value={database} onChange={(e) => setDatabase(e.target.value)} placeholder="database" />
            </Box>
            <Box flex="1" minW="160px">
              <Text fontWeight="medium" mb={1}>Schema</Text>
              <Input value={schema} onChange={(e) => setSchema(e.target.value)} placeholder="schema" />
            </Box>
            <Box flex="1" minW="160px">
              <Text fontWeight="medium" mb={1}>Table</Text>
              <Input value={table} onChange={(e) => setTable(e.target.value)} placeholder="table" />
            </Box>
            <Box minW="140px">
              <Text fontWeight="medium" mb={1}>На странице</Text>
              <Select value={limit} onChange={(e) => setLimit(Number(e.target.value))}>
                <option value={20}>20</option>
                <option value={50}>50</option>
                <option value={100}>100</option>
              </Select>
            </Box>
            <HStack spacing={3}>
              <Button leftIcon={<FiRefreshCw />} colorScheme="cyan" onClick={() => { setOffset(0); fetchGroups(); }}>
                Применить
              </Button>
              <Button variant="ghost" onClick={handleResetFilters}>Сбросить</Button>
            </HStack>
          </Stack>
        </CardBody>
      </Card>

      {error && (
        <Box mb={4} color="red.400">{error}</Box>
      )}

      {loading ? (
        <Flex justify="center" mt={10}>
          <Spinner size="xl" />
        </Flex>
      ) : groups.length === 0 ? (
        <Card variant="surface">
          <CardBody>
            <Heading size="sm" mb={2}>Группы не найдены</Heading>
            <Text color="text.muted">Попробуй изменить фильтры или обновить страницу.</Text>
          </CardBody>
        </Card>
      ) : (
        <Card variant="surface">
          <CardBody>
            <Table variant="dataGrid" size="sm">
              <Thead>
                <Tr>
                  <Th>ID</Th>
                  <Th>View</Th>
                  <Th>Источник</Th>
                  <Th>Схема</Th>
                  <Th>Таблица</Th>
                  <Th>Статус</Th>
                  <Th>Создано</Th>
                  <Th>Решено</Th>
                  <Th></Th>
                </Tr>
              </Thead>
              <Tbody>
                {groups.map((g) => (
                  <Tr key={g.id}>
                    <Td>#{g.id}</Td>
                    <Td>{g.schema_id}</Td>
                    <Td>
                      <HStack spacing={2}>
                        <Icon as={FiDatabase} />
                        <Text>{g.database_name}</Text>
                      </HStack>
                    </Td>
                    <Td>{g.schema_name}</Td>
                    <Td>{g.table_name}</Td>
                    <Td>
                      <Badge colorScheme={g.status === 'open' ? 'yellow' : 'green'} textTransform="none">{g.status}</Badge>
                    </Td>
                    <Td>{formatDate(g.created_at)}</Td>
                    <Td>{formatDate(g.resolved_at)}</Td>
                    <Td textAlign="right">
                      <Button size="sm" variant="outline" onClick={() => navigate(`/column-mismatches/${g.id}`)}>
                        Открыть
                      </Button>
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>

            <Flex justify="space-between" align="center" mt={4}>
              <Text color="text.muted">Страница {page}</Text>
              <HStack spacing={3}>
                <Button onClick={() => setOffset(Math.max(0, offset - limit))} isDisabled={offset === 0}>
                  Назад
                </Button>
                <Button onClick={() => setOffset(offset + limit)} isDisabled={groups.length < limit}>
                  Вперёд
                </Button>
              </HStack>
            </Flex>
          </CardBody>
        </Card>
      )}
    </Box>
  );
};

export default ColumnMismatchListPage;
