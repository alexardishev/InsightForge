import React, { useEffect, useMemo, useState } from 'react';
import {
  Badge,
  Box,
  Button,
  Checkbox,
  Flex,
  Heading,
  Radio,
  RadioGroup,
  Spinner,
  Stack,
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
import { useNavigate, useParams } from 'react-router-dom';
import { useHttp } from '../../hooks/http.hook';
import { ColumnMismatchGroup } from './ColumnMismatchListPage';

export interface ColumnMismatchItem {
  id: number;
  group_id: number;
  old_column_name?: string | null;
  new_column_name?: string | null;
  score?: number | null;
  type: 'schema_only' | 'missing_in_dwh' | 'dwh_only' | 'rename_candidate';
}

interface ColumnMismatchGroupResponse {
  group: ColumnMismatchGroup;
  items: ColumnMismatchItem[];
}

type ActionChoice = 'none' | 'delete' | 'ignore';

type RenameKey = `${string}->${string}`;

const ColumnMismatchDetailsPage: React.FC = () => {
  const { id } = useParams();
  const { request, loading } = useHttp();
  const toast = useToast();
  const navigate = useNavigate();

  const [group, setGroup] = useState<ColumnMismatchGroup | null>(null);
  const [items, setItems] = useState<ColumnMismatchItem[]>([]);
  const [schemaActions, setSchemaActions] = useState<Record<string, ActionChoice>>({});
  const [dwhActions, setDwhActions] = useState<Record<string, ActionChoice>>({});
  const [missingActions, setMissingActions] = useState<Record<string, ActionChoice>>({});
  const [selectedRenames, setSelectedRenames] = useState<Set<RenameKey>>(new Set());
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isResolved = group?.status === 'resolved';

  const sectionBg = useColorModeValue({
    schema_only: 'yellow.50',
    dwh_only: 'yellow.50',
    missing_in_dwh: 'blue.50',
    rename_candidate: 'green.50',
  }, {
    schema_only: 'yellow.900',
    dwh_only: 'yellow.900',
    missing_in_dwh: 'blue.900',
    rename_candidate: 'green.900',
  });

  const fetchGroup = async () => {
    if (!id) return;
    setError(null);
    try {
      const data = await request<ColumnMismatchGroupResponse>(`/api/column-mismatch-groups/${id}`);
      setGroup(data.group);
      setItems(data.items || []);
      setSchemaActions({});
      setDwhActions({});
      setMissingActions({});
      setSelectedRenames(new Set());
    } catch (e) {
      console.error(e);
      setError('Не удалось загрузить группу рассинхронов');
      toast({ title: 'Ошибка загрузки', status: 'error', duration: 3000, isClosable: true });
    }
  };

  useEffect(() => {
    fetchGroup();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  const renameCandidates = useMemo(() => items.filter((i) => i.type === 'rename_candidate'), [items]);
  const schemaOnly = useMemo(() => items.filter((i) => i.type === 'schema_only'), [items]);
  const dwhOnly = useMemo(() => items.filter((i) => i.type === 'dwh_only'), [items]);
  const missingInDwh = useMemo(() => items.filter((i) => i.type === 'missing_in_dwh'), [items]);

  const toggleRename = (oldName?: string | null, newName?: string | null) => {
    if (!oldName || !newName) return;
    const key: RenameKey = `${oldName}->${newName}`;
    setSelectedRenames((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const handleActionChange = (name: string, value: ActionChoice, setter: React.Dispatch<React.SetStateAction<Record<string, ActionChoice>>>) => {
    setter((prev) => ({ ...prev, [name]: value }));
  };

  const buildResolution = () => {
    const deletes = new Set<string>();
    const ignores = new Set<string>();
    const renames: { old_name: string; new_name: string }[] = [];

    selectedRenames.forEach((key) => {
      const [oldName, newName] = key.split('->');
      renames.push({ old_name: oldName, new_name: newName });
    });

    Object.entries(schemaActions).forEach(([name, action]) => {
      if (action === 'delete') deletes.add(name);
      if (action === 'ignore') ignores.add(name);
    });

    Object.entries(dwhActions).forEach(([name, action]) => {
      if (action === 'delete') deletes.add(name);
      if (action === 'ignore') ignores.add(name);
    });

    Object.entries(missingActions).forEach(([name, action]) => {
      if (action === 'ignore') ignores.add(name);
    });

    return {
      renames,
      deletes: Array.from(deletes),
      ignores: Array.from(ignores),
    };
  };

  const canApply = useMemo(() => {
    if (isResolved) return false;
    const res = buildResolution();
    return res.renames.length > 0 || res.deletes.length > 0 || res.ignores.length > 0;
  }, [dwhActions, isResolved, missingActions, schemaActions, selectedRenames]);

  const handleApply = async () => {
    if (!id) return;
    const resolution = buildResolution();
    setSubmitting(true);
    setError(null);
    try {
      await request(`/api/column-mismatch-groups/${id}/apply`, 'POST', resolution);
      toast({ title: 'Решение применено', status: 'success', duration: 3000, isClosable: true });
      await fetchGroup();
    } catch (e) {
      console.error(e);
      setError('Не удалось применить решение');
      toast({ title: 'Ошибка применения', status: 'error', duration: 3000, isClosable: true });
    } finally {
      setSubmitting(false);
    }
  };

  const renderActionTable = (
    title: string,
    itemsToRender: ColumnMismatchItem[],
    type: ColumnMismatchItem['type'],
    actions: Record<string, ActionChoice>,
    setter: React.Dispatch<React.SetStateAction<Record<string, ActionChoice>>>,
    showDelete = true,
    disabled = false,
  ) => (
    <Box borderWidth="1px" borderRadius="lg" p={4} bg={sectionBg[type]}>
      <Heading size="md" mb={3}>{title}</Heading>
      {itemsToRender.length === 0 ? (
        <Text>Нет элементов.</Text>
      ) : (
        <Table size="sm" variant="simple">
          <Thead>
            <Tr>
              <Th>Column</Th>
              <Th>Действие</Th>
            </Tr>
          </Thead>
          <Tbody>
            {itemsToRender.map((item) => {
              const name = item.old_column_name || item.new_column_name || '-';
              const currentAction = actions[name] || 'none';
              return (
                <Tr key={`${item.id}-${name}`}>
                  <Td>{name}</Td>
                  <Td>
                    <RadioGroup
                      value={currentAction}
                      onChange={(val) => handleActionChange(name, val as ActionChoice, setter)}
                      isDisabled={disabled}
                    >
                      <Stack direction="row">
                        <Radio value="none">Ничего</Radio>
                        {showDelete && <Radio value="delete">Удалить из view</Radio>}
                        <Radio value="ignore">Игнорировать</Radio>
                      </Stack>
                    </RadioGroup>
                  </Td>
                </Tr>
              );
            })}
          </Tbody>
        </Table>
      )}
    </Box>
  );

  const renderRenameCandidates = () => (
    <Box borderWidth="1px" borderRadius="lg" p={4} bg={sectionBg.rename_candidate}>
      <Heading size="md" mb={3}>Кандидаты на переименование</Heading>
      {renameCandidates.length === 0 ? (
        <Text>Нет кандидатов.</Text>
      ) : (
        <Table size="sm" variant="simple">
          <Thead>
            <Tr>
              <Th>Old column</Th>
              <Th>New column</Th>
              <Th>Score</Th>
              <Th>Выбрать</Th>
            </Tr>
          </Thead>
          <Tbody>
            {renameCandidates.map((item) => {
              const key: RenameKey = `${item.old_column_name || ''}->${item.new_column_name || ''}`;
              const selected = selectedRenames.has(key);
              const scoreDisplay = item.score ? Math.round((item.score || 0) * 100) / 100 : undefined;
              return (
                <Tr key={item.id}>
                  <Td>{item.old_column_name}</Td>
                  <Td>{item.new_column_name}</Td>
                  <Td>{scoreDisplay !== undefined ? scoreDisplay : '-'}</Td>
                  <Td>
                    <Checkbox
                      isChecked={selected}
                      onChange={() => toggleRename(item.old_column_name, item.new_column_name)}
                      isDisabled={isResolved}
                    />
                  </Td>
                </Tr>
              );
            })}
          </Tbody>
        </Table>
      )}
    </Box>
  );

  if (loading && !group) {
    return (
      <Flex justify="center" mt={10}>
        <Spinner size="xl" />
      </Flex>
    );
  }

  return (
    <Box p={8} maxW="1200px" mx="auto">
      <Button mb={4} variant="ghost" onClick={() => navigate('/column-mismatches')}>
        ← Назад к списку
      </Button>

      {error && (
        <Box mb={4} color="red.500">{error}</Box>
      )}

      {group && (
        <Box borderWidth="1px" borderRadius="lg" p={4} mb={6}>
          <Heading size="lg" mb={3}>Группа #{group.id}</Heading>
          <Flex gap={4} wrap="wrap" mb={3}>
            <Badge colorScheme={group.status === 'open' ? 'yellow' : 'green'}>{group.status}</Badge>
            <Text>View ID: {group.schema_id}</Text>
            <Text>DB: {group.database_name}</Text>
            <Text>Schema: {group.schema_name}</Text>
            <Text>Table: {group.table_name}</Text>
          </Flex>
          <Text>Создано: {new Date(group.created_at).toLocaleString()}</Text>
          {group.resolved_at && <Text>Закрыто: {new Date(group.resolved_at).toLocaleString()}</Text>}
        </Box>
      )}

      {isResolved && (
        <Box mb={4} color="gray.600">Эта группа уже закрыта. Изменения недоступны.</Box>
      )}

      <Stack spacing={6}>
        {renderActionTable('Колонки есть в схеме, но нет в OLTP (schema_only)', schemaOnly, 'schema_only', schemaActions, setSchemaActions, true, isResolved)}
        {renderActionTable('Колонки есть в схеме и OLTP, но отсутствуют в DWH (missing_in_dwh)', missingInDwh, 'missing_in_dwh', missingActions, setMissingActions, false, isResolved)}
        {renderActionTable('Колонки есть в DWH, но не описаны в схеме (dwh_only)', dwhOnly, 'dwh_only', dwhActions, setDwhActions, true, isResolved)}
        {renderRenameCandidates()}
      </Stack>

      <Flex justify="flex-end" mt={6} gap={3}>
        <Button colorScheme="blue" onClick={handleApply} isDisabled={!canApply || submitting} isLoading={submitting}>
          Применить решение
        </Button>
      </Flex>
    </Box>
  );
};

export default ColumnMismatchDetailsPage;
