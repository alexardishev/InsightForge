import React, { useState, useEffect } from 'react';
import { Box, Heading, VStack, Text, HStack, Badge, Divider } from '@chakra-ui/react';
import { useSelector, useDispatch } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import type { RootState, AppDispatch } from '../../app/store';
import {
  setCurrentDb,
  setCurrentSchema,
  toggleTable,
  toggleColumn,
  setTableColumns,
  setColumnAlias,
} from './viewBuilderSlice';
import DatabaseSelector from './components/DatabaseSelector';
import SchemaSelector from './components/SchemaSelector';
import TableSelector from './components/TableSelector';
import ColumnsGrid from './components/ColumnsGrid';
import { useHttp } from '../../hooks/http.hook';
import { appendTables } from '../settings/settingsSlice';
import FlowLayout from '../../components/FlowLayout';
import { builderSteps } from './flowSteps';

const ViewBuilderPage: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const navigate = useNavigate();
  const data = useSelector((state: RootState) => state.settings.dataBaseInfo);
  const selectedConnections = useSelector(
    (state: RootState) => state.settings.selectedConnections,
  );
  const { currentDb, currentSchema, selectedSources } = useSelector(
    (state: RootState) => state.viewBuilder,
  );
  const { request } = useHttp();
  const url = '/api';

  const [tablesState, setTablesState] = useState<any[]>([]);
  const [page, setPage] = useState(1);
  const pageSize = 20;


  const handleToggleTable = (table: string) => {
    if (!currentDb || !currentSchema) return;
    dispatch(toggleTable({ db: currentDb, schema: currentSchema, table }));
  };

  const handleToggleColumn = (
    table: string,
    column: any,
  ) => {
    if (!currentDb || !currentSchema) return;
    dispatch(
      toggleColumn({
        db: currentDb,
        schema: currentSchema,
        table,
        column: column.name,
        isPrimaryKey: column.is_primary_key || column.is_pk,
        isUpdateKey: column.is_update_key,
      }),
    );
  };

  const handleSetTableColumns = (table: string, columns: any[]) => {
    if (!currentDb || !currentSchema) return;
    dispatch(
      setTableColumns({
        db: currentDb,
        schema: currentSchema,
        table,
        columns: columns.map((col) => ({
          name: col.name,
          isPrimaryKey: col.is_primary_key || col.is_pk,
          isUpdateKey: col.is_update_key,
        })),
      }),
    );
  };

  const handleAliasChange = (table: string, column: string, alias: string) => {
    if (!currentDb || !currentSchema) return;
    dispatch(setColumnAlias({ db: currentDb, schema: currentSchema, table, column, alias }));
  };

  const selectedDatabase = data?.find((db: any) => db.name === currentDb);
  const selectedSchemaData = selectedDatabase?.schemas?.find((schema: any) => schema.name === currentSchema);

  const currentSelection = selectedSources.find(
    (source) => source.db === currentDb && source.schema === currentSchema,
  );
  const selectedTables = currentSelection?.selectedTables ?? [];
  const selectedColumns = currentSelection?.selectedColumns ?? [];

  useEffect(() => {
    setTablesState(selectedSchemaData?.tables || []);
    setPage(1);
  }, [selectedSchemaData]);

  const schemaDataWithTables = selectedSchemaData
    ? { ...selectedSchemaData, tables: tablesState }
    : undefined;

  const loadMore = async () => {
    const nextPage = page + 1;
    try {
      const body = {
        connection_strings: selectedConnections.map((connection_string) => ({
          connection_string,
        })),
        page: nextPage,
        page_size: pageSize,
      };
      const dbInfo = await request(`${url}/get-db`, 'POST', body);
      const db = dbInfo.find((d: any) => d.name === currentDb);
      const schema = db?.schemas?.find((s: any) => s.name === currentSchema);
      const newTables = schema?.tables || [];
      if (newTables.length > 0) {
        setTablesState(prev => [...prev, ...newTables]);
        dispatch(appendTables({ db: currentDb, schema: currentSchema, tables: newTables }));
        setPage(nextPage);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const handleBuildView = () => {
    navigate('/joins');
  };

  return (
    <FlowLayout
      steps={builderSteps}
      currentStep={1}
      onBack={() => navigate('/settings')}
      onNext={handleBuildView}
      primaryLabel="Перейти к джоинам"
      secondaryLabel="К подключениям"
      isNextDisabled={selectedSources.every((source) => source.selectedColumns.length === 0)}
    >
      <VStack align="stretch" spacing={6}>
        <Box>
          <Heading size="lg">Tables & Columns</Heading>
          <Text color="text.muted" mt={2}>
            Выбери схемы, таблицы и колонки. Безопасные дефолты помогают избежать случайных
            пропусков ключей.
          </Text>
          <HStack spacing={3} mt={3} color="text.muted">
            <Badge colorScheme="cyan">Source</Badge>
            <Badge colorScheme="purple">Schema</Badge>
            <Badge colorScheme="green">Columns</Badge>
          </HStack>
        </Box>

        <Divider borderColor="border.subtle" />

        <Box>
          <DatabaseSelector
            data={data}
            selectedDb={currentDb}
            onChange={(db) => dispatch(setCurrentDb(db))}
          />
        </Box>

        {currentDb && selectedDatabase && (
          <Box>
            <SchemaSelector
              selectedDatabase={selectedDatabase}
              selectedSchema={currentSchema}
              onChange={(schema) => dispatch(setCurrentSchema(schema))}
            />
          </Box>
        )}

        {currentSchema && schemaDataWithTables && (
          <TableSelector
            selectedSchemaData={schemaDataWithTables}
            selectedTables={selectedTables}
            onToggleTable={handleToggleTable}
            onLoadMore={loadMore}
          />
        )}

        <ColumnsGrid
          selectedTables={selectedTables}
          selectedSchemaData={schemaDataWithTables}
          selectedColumns={selectedColumns}
          onToggleColumn={handleToggleColumn}
          onSetTableColumns={handleSetTableColumns}
          onAliasChange={handleAliasChange}
        />

        {selectedSources.every((source) => source.selectedColumns.length === 0) && (
          <Box
            p={4}
            border="1px dashed"
            borderColor="border.subtle"
            borderRadius="lg"
            textAlign="center"
            color="text.muted"
          >
            Добавь минимум одну колонку, чтобы перейти к настройке джоинов.
          </Box>
        )}
      </VStack>
    </FlowLayout>
  );
};

export default ViewBuilderPage;
