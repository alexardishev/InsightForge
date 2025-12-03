import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Badge,
  Box,
  Button,
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
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import { useNavigate } from 'react-router-dom';
import { useHttp } from '../../hooks/http.hook';

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

  const headerBg = useColorModeValue('gray.50', 'gray.800');
  const rowHoverBg = useColorModeValue('gray.100', 'gray.700');

  const formatDate = (value?: string | null) => {
    if (!value) return '-';
    return new Date(value).toLocaleString();
  };

  return (
    <Box p={8} maxW="1400px" mx="auto">
      <Flex justify="space-between" align="center" mb={6} wrap="wrap" gap={4}>
        <Heading size="lg">Группы рассинхронов колонок</Heading>
      </Flex>

      <Box borderWidth="1px" borderRadius="lg" p={4} mb={6}>
        <Flex gap={4} wrap="wrap">
          <Box>
            <Text fontWeight="medium" mb={1}>Статус</Text>
            <Select value={status} onChange={(e) => setStatus(e.target.value as typeof status)} width="200px">
              <option value="open">Открытые</option>
              <option value="resolved">Закрытые</option>
              <option value="all">Все</option>
            </Select>
          </Box>
          <Box>
            <Text fontWeight="medium" mb={1}>Database</Text>
            <Input value={database} onChange={(e) => setDatabase(e.target.value)} placeholder="database" width="200px" />
          </Box>
          <Box>
            <Text fontWeight="medium" mb={1}>Schema</Text>
            <Input value={schema} onChange={(e) => setSchema(e.target.value)} placeholder="schema" width="200px" />
          </Box>
          <Box>
            <Text fontWeight="medium" mb={1}>Table</Text>
            <Input value={table} onChange={(e) => setTable(e.target.value)} placeholder="table" width="200px" />
          </Box>
          <Box>
            <Text fontWeight="medium" mb={1}>На странице</Text>
            <Select value={limit} onChange={(e) => setLimit(Number(e.target.value))} width="120px">
              <option value={20}>20</option>
              <option value={50}>50</option>
              <option value={100}>100</option>
            </Select>
          </Box>
        </Flex>

        <Flex mt={4} gap={3}>
          <Button colorScheme="blue" onClick={() => { setOffset(0); fetchGroups(); }}>Применить</Button>
          <Button variant="ghost" onClick={handleResetFilters}>Сбросить</Button>
        </Flex>
      </Box>

      {error && (
        <Box mb={4} color="red.500">
          {error}
        </Box>
      )}

      {loading ? (
        <Flex justify="center" mt={10}>
          <Spinner size="xl" />
        </Flex>
      ) : groups.length === 0 ? (
        <Text mt={6}>Группы рассинхронов не найдены.</Text>
      ) : (
        <Box borderWidth="1px" borderRadius="lg" overflowX="auto" shadow="sm">
          <Table variant="simple" size="sm">
            <Thead bg={headerBg}>
              <Tr>
                <Th>ID</Th>
                <Th>View ID</Th>
                <Th>Database</Th>
                <Th>Schema</Th>
                <Th>Table</Th>
                <Th>Status</Th>
                <Th>Created</Th>
                <Th>Resolved</Th>
                <Th></Th>
              </Tr>
            </Thead>
            <Tbody>
              {groups.map((g) => (
                <Tr key={g.id} _hover={{ bg: rowHoverBg }}>
                  <Td>{g.id}</Td>
                  <Td>{g.schema_id}</Td>
                  <Td>{g.database_name}</Td>
                  <Td>{g.schema_name}</Td>
                  <Td>{g.table_name}</Td>
                  <Td>
                    <Badge colorScheme={g.status === 'open' ? 'yellow' : 'green'} textTransform="none">{g.status}</Badge>
                  </Td>
                  <Td>{formatDate(g.created_at)}</Td>
                  <Td>{formatDate(g.resolved_at)}</Td>
                  <Td>
                    <Button size="sm" colorScheme="blue" onClick={() => navigate(`/column-mismatches/${g.id}`)}>
                      Открыть
                    </Button>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </Box>
      )}

      <Flex justify="space-between" align="center" mt={6}>
        <Text>Страница {page} (offset {offset})</Text>
        <Flex gap={3}>
          <Button onClick={() => setOffset(Math.max(0, offset - limit))} isDisabled={offset === 0}>Назад</Button>
          <Button onClick={() => setOffset(offset + limit)} isDisabled={groups.length < limit}>Вперёд</Button>
        </Flex>
      </Flex>
    </Box>
  );
};

export default ColumnMismatchListPage;
