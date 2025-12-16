import React, { useState } from 'react';
import {
  Box,
  Heading,
  VStack,
  HStack,
  Select,
  Button,
  Text,
  Input,
  IconButton,
  Alert,
  AlertIcon,
  Tag,
  Divider,
  SimpleGrid,
  Badge,
  Card,
  CardBody,
  Tooltip,
  Icon,
} from '@chakra-ui/react';
import { useDispatch, useSelector } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import type { RootState, AppDispatch } from '../../app/store';
import { addJoin, removeJoin, setViewName } from './viewBuilderSlice';
import { DeleteIcon } from '@chakra-ui/icons';
import { FiKey, FiLink2 } from 'react-icons/fi';
import FlowLayout from '../../components/FlowLayout';
import { builderSteps } from './flowSteps';

interface Column {
  name: string;
}

const JoinBuilderPage: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const navigate = useNavigate();

  const data = useSelector((state: RootState) => state.settings.dataBaseInfo);
  const {
    selectedDb,
    selectedSchema,
    selectedTables,
    selectedColumns,
    joins,
    viewName,
  } = useSelector((state: RootState) => state.viewBuilder);

  const selectedDatabase = data?.find((db: any) => db.name === selectedDb);
  const selectedSchemaData = selectedDatabase?.schemas?.find((s: any) => s.name === selectedSchema);

  const [mainTable, setMainTable] = useState('');
  const [joinTable, setJoinTable] = useState('');
  const [mainColumn, setMainColumn] = useState('');
  const [joinColumn, setJoinColumn] = useState('');

  const requiresJoin = selectedTables.length > 1;
  const cartesianRisk = requiresJoin && joins.length === 0;
  const canProceed = !requiresJoin || joins.length > 0;
  const columnsByTable = (tableName: string): Column[] =>
    selectedSchemaData?.tables.find((t: any) => t.name === tableName)?.columns ?? [];

  const handleAddJoin = () => {
    if (!mainTable || !joinTable || !mainColumn || !joinColumn) return;
    const join = {
      inner: {
        source: selectedDb,
        schema: selectedSchema,
        table: joinTable,
        main_table: mainTable,
        column_first: mainColumn,
        column_second: joinColumn,
      },
    };
    dispatch(addJoin(join));
    setJoinTable('');
    setMainColumn('');
    setJoinColumn('');
  };

  const handleNext = () => {
    navigate('/transforms');
  };

  return (
    <FlowLayout
      steps={builderSteps}
      currentStep={2}
      onBack={() => navigate('/builder')}
      onNext={handleNext}
      primaryLabel="К трансформациям"
      secondaryLabel="Назад к таблицам"
      isNextDisabled={!canProceed}
    >
      <VStack spacing={6} align="stretch">
        <Box>
          <Heading size="lg">Join Builder</Heading>
          <Text color="text.muted" mt={2}>
            Настрой добавление таблиц, исключая cartesian join. Заполни имя витрины и ключи для
            соединения.
          </Text>
        </Box>

        <Alert status={cartesianRisk ? 'warning' : 'info'} borderRadius="md" variant="left-accent">
          <AlertIcon />
          {cartesianRisk
            ? 'Добавь хотя бы один join для нескольких таблиц — иначе возможен cartesian join.'
            : 'Для одной таблицы джоины не обязательны. Мы подсветим рискованные связки.'}
        </Alert>

        <Card variant="surface">
          <CardBody>
            <HStack justify="space-between" mb={3} align="center">
              <Text fontWeight="semibold">Состав витрины</Text>
              <Badge colorScheme="cyan">{selectedTables.length} таблиц</Badge>
            </HStack>
            <SimpleGrid columns={{ base: 1, md: 2 }} spacing={3}>
              {selectedTables.map((table) => {
                const cols = columnsByTable(table) as Column[];
                return (
                  <Box
                    key={table}
                    p={3}
                    border="1px solid"
                    borderColor="border.subtle"
                    borderRadius="lg"
                    bg="bg.elevated"
                  >
                    <HStack justify="space-between" mb={2}>
                      <HStack spacing={2}>
                        <Icon as={FiLink2} color="accent.primary" />
                        <Text fontWeight="bold">{table}</Text>
                      </HStack>
                      <Tag colorScheme="purple">{cols.length} колонок</Tag>
                    </HStack>
                    <HStack spacing={2} flexWrap="wrap">
                      {selectedColumns
                        .filter((c) => c.table === table)
                        .slice(0, 6)
                        .map((c: { column: string }) => (
                          <Tag key={c.column} colorScheme="cyan" variant="subtle">
                            {c.column}
                          </Tag>
                        ))}
                      {selectedColumns.filter((c) => c.table === table).length > 6 && (
                        <Tag variant="outline">…</Tag>
                      )}
                    </HStack>
                  </Box>
                );
              })}
            </SimpleGrid>
          </CardBody>
        </Card>

        <Input
          placeholder="Имя витрины"
          value={viewName}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => dispatch(setViewName(e.target.value))}
        />

        <Divider borderColor="border.subtle" />

        <Card variant="surface">
          <CardBody>
            <HStack justify="space-between" mb={2} align="center">
              <Text fontWeight="semibold">Настрой ключи join</Text>
              <Badge colorScheme={canProceed ? 'green' : 'yellow'}>
                {canProceed ? 'готово к переходу' : 'нужно правило'}
              </Badge>
            </HStack>
            <Text color="text.muted" fontSize="sm" mb={4}>
              Выбирай пары таблиц и колонок. Мы блокируем риск картезианских джоинов и подсвечиваем незаполненные поля.
            </Text>
            <SimpleGrid columns={{ base: 1, md: 2 }} spacing={3}>
              <Select
                placeholder="Основная таблица"
                value={mainTable}
                variant="filled"
                onChange={(e: React.ChangeEvent<HTMLSelectElement>) => setMainTable(e.target.value)}
              >
                {selectedTables.map((t: string) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </Select>
              <Select
                placeholder="Колонка"
                value={mainColumn}
                variant="filled"
                onChange={(e: React.ChangeEvent<HTMLSelectElement>) => setMainColumn(e.target.value)}
              >
                {columnsByTable(mainTable).map((col: Column) => (
                  <option key={col.name} value={col.name}>
                    {col.name}
                  </option>
                ))}
              </Select>
              <Select
                placeholder="Таблица для join"
                value={joinTable}
                variant="filled"
                onChange={(e: React.ChangeEvent<HTMLSelectElement>) => setJoinTable(e.target.value)}
              >
                {selectedTables.map((t: string) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </Select>
              <Select
                placeholder="Колонка"
                value={joinColumn}
                variant="filled"
                onChange={(e: React.ChangeEvent<HTMLSelectElement>) => setJoinColumn(e.target.value)}
              >
                {columnsByTable(joinTable).map((col: Column) => (
                  <option key={col.name} value={col.name}>
                    {col.name}
                  </option>
                ))}
              </Select>
            </SimpleGrid>
            <Button onClick={handleAddJoin} variant="glow" alignSelf="flex-start" mt={4}>
              Добавить join
            </Button>
          </CardBody>
        </Card>

        <Card variant="surface">
          <CardBody>
            <HStack justify="space-between" mb={3}>
              <Text fontWeight="semibold">Карта джоинов</Text>
              <Badge colorScheme={joins.length ? 'green' : 'yellow'}>
                {joins.length ? `${joins.length} правил` : 'правил пока нет'}
              </Badge>
            </HStack>
            {joins.length === 0 ? (
              <Text color="text.muted">Добавь хотя бы одно правило, если используешь более одной таблицы.</Text>
            ) : (
              <SimpleGrid columns={{ base: 1, md: 2 }} spacing={3}>
                {joins.map((j: any, idx: number) => (
                  <Card key={`${j.inner.main_table}-${idx}`} variant="glass">
                    <CardBody>
                      <HStack justify="space-between" mb={2}>
                        <HStack>
                          <Icon as={FiLink2} />
                          <Text fontWeight="bold">INNER JOIN</Text>
                        </HStack>
                        <IconButton
                          aria-label="delete"
                          icon={<DeleteIcon />}
                          size="sm"
                          onClick={() => dispatch(removeJoin(idx))}
                          variant="ghost"
                        />
                      </HStack>
                      <VStack align="stretch" spacing={2} fontSize="sm">
                        <HStack>
                          <Tag colorScheme="cyan">{j.inner.main_table}</Tag>
                          <Icon as={FiKey} />
                          <Text>{j.inner.column_first}</Text>
                        </HStack>
                        <HStack>
                          <Tag colorScheme="purple">{j.inner.table}</Tag>
                          <Icon as={FiKey} />
                          <Text>{j.inner.column_second}</Text>
                        </HStack>
                        <Tooltip label="Мы блокируем cartesian join, если ключ не указан" placement="top">
                          <Text color="text.muted">Ключи обязательно должны быть выбраны.</Text>
                        </Tooltip>
                      </VStack>
                    </CardBody>
                  </Card>
                ))}
              </SimpleGrid>
            )}
          </CardBody>
        </Card>
      </VStack>
    </FlowLayout>
  );
};

export default JoinBuilderPage;
