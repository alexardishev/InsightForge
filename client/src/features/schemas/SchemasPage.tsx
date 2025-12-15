import React, { useEffect, useState } from 'react';
import {
  Box,
  Button,
  Heading,
  HStack,
  Spinner,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useToast,
} from '@chakra-ui/react';
import { useHttp } from '../../hooks/http.hook';

type SchemaInfo = {
  id: number;
  name: string;
  source_count: number;
  table_count: number;
};

const SchemasPage: React.FC = () => {
  const { request, loading } = useHttp();
  const toast = useToast();
  const [schemas, setSchemas] = useState<SchemaInfo[]>([]);
  const [etlLoading, setEtlLoading] = useState<Record<number, boolean>>({});

  const fetchSchemas = async () => {
    try {
      const data = await request<SchemaInfo[]>('/api/schemas');
      setSchemas(data);
    } catch (error) {
      console.error(error);
      toast({
        title: 'Не удалось загрузить схемы',
        description: 'Попробуйте обновить страницу',
        status: 'error',
        duration: 5000,
        isClosable: true,
      });
    }
  };

  const startEtl = async (schema: SchemaInfo) => {
    setEtlLoading((prev) => ({ ...prev, [schema.id]: true }));
    try {
      await request(`/api/schemas/${schema.id}/etl`, 'POST');
      toast({
        title: 'ETL запущен',
        description: `Задача для схемы «${schema.name}» запущена. Ожидайте уведомления о завершении.`,
        status: 'success',
        duration: 6000,
        isClosable: true,
      });
    } catch (error) {
      console.error(error);
      toast({
        title: 'Ошибка запуска ETL',
        description: 'Попробуйте позже или проверьте подключение.',
        status: 'error',
        duration: 5000,
        isClosable: true,
      });
    } finally {
      setEtlLoading((prev) => ({ ...prev, [schema.id]: false }));
    }
  };

  useEffect(() => {
    fetchSchemas();
  }, []);

  return (
    <Box p={8} maxW="1200px" mx="auto">
      <HStack justify="space-between" mb={4} align="center">
        <Heading size="lg">Схемы</Heading>
        <Button onClick={fetchSchemas} isLoading={loading} variant="outline">
          Обновить
        </Button>
      </HStack>

      {loading && schemas.length === 0 ? (
        <HStack justify="center" mt={10}>
          <Spinner size="xl" />
          <Text>Загрузка схем...</Text>
        </HStack>
      ) : (
        <Table variant="simple" size="md">
          <Thead>
            <Tr>
              <Th>ID</Th>
              <Th>Название</Th>
              <Th>Источники</Th>
              <Th>Таблицы</Th>
              <Th textAlign="right">Действия</Th>
            </Tr>
          </Thead>
          <Tbody>
            {schemas.length === 0 ? (
              <Tr>
                <Td colSpan={5} textAlign="center" py={6}>
                  Сохранённые схемы не найдены.
                </Td>
              </Tr>
            ) : (
              schemas.map((schema) => (
                <Tr key={schema.id}>
                  <Td>{schema.id}</Td>
                  <Td>{schema.name || 'Без названия'}</Td>
                  <Td>{schema.source_count}</Td>
                  <Td>{schema.table_count}</Td>
                  <Td textAlign="right">
                    <Button
                      colorScheme="blue"
                      onClick={() => startEtl(schema)}
                      isLoading={etlLoading[schema.id]}
                    >
                      Запустить ETL
                    </Button>
                  </Td>
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      )}
    </Box>
  );
};

export default SchemasPage;
