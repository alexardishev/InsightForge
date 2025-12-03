import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Box,
  Button,
  Flex,
  Heading,
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
import { useHttp } from '../../hooks/http.hook';

export interface ColumnRenameSuggestion {
  id: number;
  schema_id: number;
  database_name: string;
  schema_name: string;
  table_name: string;
  old_column_name: string;
  new_column_name: string;
  strategy: string;
  task_number?: string | null;
  created_at: string;
}

interface ColumnRenameSuggestionsResponse {
  items: ColumnRenameSuggestion[];
  limit: number;
  offset: number;
}

const ColumnRenameSuggestionsPage: React.FC = () => {
  const { request, loading } = useHttp();
  const [suggestions, setSuggestions] = useState<ColumnRenameSuggestion[]>([]);
  const [limit, setLimit] = useState<number>(20);
  const [offset, setOffset] = useState<number>(0);
  const [sort, setSort] = useState<'created_at_desc' | 'created_at_asc'>('created_at_desc');
  const [processingId, setProcessingId] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const toast = useToast();

  const page = useMemo(() => Math.floor(offset / limit) + 1, [offset, limit]);

  const fetchSuggestions = useCallback(async () => {
    setError(null);
    const params = new URLSearchParams({
      limit: limit.toString(),
      offset: offset.toString(),
      sort,
    });

    try {
      const data = await request<ColumnRenameSuggestionsResponse>(
        `/api/column-rename-suggestions?${params.toString()}`
      );
      setSuggestions(data.items || []);
    } catch (e) {
      console.error(e);
      setError('Не удалось загрузить предложения');
    }
  }, [limit, offset, request, sort]);

  useEffect(() => {
    fetchSuggestions();
  }, [fetchSuggestions]);

  const handleAccept = async (id: number) => {
    if (!window.confirm('Вы уверены, что хотите принять это предложение?')) return;
    setProcessingId(id);
    setError(null);

    try {
      await request(`/api/column-rename-suggestions/${id}/accept`, 'POST');
      toast({ title: 'Предложение принято', status: 'success', duration: 3000, isClosable: true });
      await fetchSuggestions();
    } catch (e) {
      console.error(e);
      setError('Не удалось применить переименование');
      toast({ title: 'Ошибка при применении', status: 'error', duration: 3000, isClosable: true });
    } finally {
      setProcessingId(null);
    }
  };

  const handleReject = async (id: number) => {
    if (!window.confirm('Вы уверены, что хотите отклонить это предложение?')) return;
    setProcessingId(id);
    setError(null);

    try {
      await request(`/api/column-rename-suggestions/${id}/reject`, 'POST');
      toast({ title: 'Предложение отклонено', status: 'info', duration: 3000, isClosable: true });
      await fetchSuggestions();
    } catch (e) {
      console.error(e);
      setError('Не удалось отклонить предложение');
      toast({ title: 'Ошибка при отклонении', status: 'error', duration: 3000, isClosable: true });
    } finally {
      setProcessingId(null);
    }
  };

  const isNextDisabled = suggestions.length < limit;

  const headerBg = useColorModeValue('gray.50', 'gray.800');
  const rowHoverBg = useColorModeValue('gray.100', 'gray.700');

  return (
    <Box p={8} maxW="1400px" mx="auto">
      <Flex justify="space-between" align="center" mb={6} wrap="wrap" gap={4}>
        <Heading size="lg">Предложения по переименованию колонок</Heading>

        <Flex gap={3} align="center">
          <Flex align="center" gap={2}>
            <Text fontWeight="medium">Сортировка:</Text>
            <Select value={sort} onChange={(e) => setSort(e.target.value as typeof sort)} width="220px">
              <option value="created_at_desc">Сначала новые</option>
              <option value="created_at_asc">Сначала старые</option>
            </Select>
          </Flex>

          <Flex align="center" gap={2}>
            <Text fontWeight="medium">На странице:</Text>
            <Select value={limit} onChange={(e) => setLimit(Number(e.target.value))} width="120px">
              <option value={20}>20</option>
              <option value={50}>50</option>
            </Select>
          </Flex>
        </Flex>
      </Flex>

      {error && (
        <Box mb={4} color="red.500">
          {error}
        </Box>
      )}

      {loading ? (
        <Flex justify="center" mt={10}>
          <Spinner size="xl" />
        </Flex>
      ) : suggestions.length === 0 ? (
        <Text mt={6}>Предложений нет.</Text>
      ) : (
        <Box borderWidth="1px" borderRadius="lg" overflowX="auto" shadow="sm">
          <Table variant="simple" size="sm">
            <Thead bg={headerBg}>
              <Tr>
                <Th>Дата</Th>
                <Th>View ID</Th>
                <Th>Database</Th>
                <Th>Schema</Th>
                <Th>Table</Th>
                <Th>Old column</Th>
                <Th>New column</Th>
                <Th>Strategy</Th>
                <Th textAlign="center">Действия</Th>
              </Tr>
            </Thead>
            <Tbody>
              {suggestions.map((s) => (
                <Tr key={s.id} _hover={{ bg: rowHoverBg }}>
                  <Td>{new Date(s.created_at).toLocaleString()}</Td>
                  <Td>{s.schema_id}</Td>
                  <Td>{s.database_name}</Td>
                  <Td>{s.schema_name}</Td>
                  <Td>{s.table_name}</Td>
                  <Td>{s.old_column_name}</Td>
                  <Td>{s.new_column_name}</Td>
                  <Td>{s.strategy}</Td>
                  <Td>
                    <Flex gap={2} justify="center">
                      <Button
                        colorScheme="green"
                        size="sm"
                        onClick={() => handleAccept(s.id)}
                        isLoading={processingId === s.id}
                      >
                        Принять
                      </Button>
                      <Button
                        colorScheme="red"
                        variant="outline"
                        size="sm"
                        onClick={() => handleReject(s.id)}
                        isLoading={processingId === s.id}
                      >
                        Отклонить
                      </Button>
                    </Flex>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </Box>
      )}

      <Flex justify="space-between" align="center" mt={6}>
        <Text>
          Страница {page} (offset {offset})
        </Text>
        <Flex gap={3}>
          <Button onClick={() => setOffset(Math.max(0, offset - limit))} isDisabled={offset === 0}>
            Назад
          </Button>
          <Button onClick={() => setOffset(offset + limit)} isDisabled={isNextDisabled}>
            Вперёд
          </Button>
        </Flex>
      </Flex>
    </Box>
  );
};

export default ColumnRenameSuggestionsPage;
