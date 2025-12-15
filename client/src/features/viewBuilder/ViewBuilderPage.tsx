import React, { useState, useEffect } from 'react';
import { Box, Heading, VStack, Text, HStack, Badge, Divider } from '@chakra-ui/react';
import { useSelector, useDispatch } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import type { RootState, AppDispatch } from '../../app/store';
import {
  setSelectedDb,
  setSelectedSchema,
  toggleTable,
  toggleColumn,
  setTableColumns,
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
  const connectionsMap = useSelector(
    (state: RootState) => state.settings.connectionsMap,
  );
  const { selectedDb, selectedSchema, selectedTables, selectedColumns } =
    useSelector((state: RootState) => state.viewBuilder);
  const { request } = useHttp();
  const url = '/api';

  const [tablesState, setTablesState] = useState<any[]>([]);
  const [page, setPage] = useState(1);
  const pageSize = 20;


  const handleToggleTable = (table: string) => {
    dispatch(toggleTable(table));
  };

  const handleToggleColumn = (
    table: string,
    column: any,
  ) => {
    dispatch(
      toggleColumn({
        table,
        column: column.name,
        isPrimaryKey: column.is_primary_key || column.is_pk,
        isUpdateKey: column.is_update_key,
      }),
    );
  };

  const handleSetTableColumns = (table: string, columns: any[]) => {
    dispatch(
      setTableColumns({
        table,
        columns: columns.map((col) => ({
          name: col.name,
          isPrimaryKey: col.is_primary_key || col.is_pk,
          isUpdateKey: col.is_update_key,
        })),
      }),
    );
  };

  const selectedDatabase = data?.find((db: any) => db.name === selectedDb);
  const selectedSchemaData = selectedDatabase?.schemas?.find((schema: any) => schema.name === selectedSchema);

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
        connection_strings: [{ connection_string: connectionsMap }],
        page: nextPage,
        page_size: pageSize,
      };
      const dbInfo = await request(`${url}/get-db`, 'POST', body);
      const db = dbInfo.find((d: any) => d.name === selectedDb);
      const schema = db?.schemas?.find((s: any) => s.name === selectedSchema);
      const newTables = schema?.tables || [];
      if (newTables.length > 0) {
        setTablesState(prev => [...prev, ...newTables]);
        dispatch(appendTables({ db: selectedDb, schema: selectedSchema, tables: newTables }));
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
      isNextDisabled={selectedColumns.length === 0}
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
            selectedDb={selectedDb}
            onChange={(db) => dispatch(setSelectedDb(db))}
          />
        </Box>

        {selectedDb && selectedDatabase && (
          <Box>
            <SchemaSelector
              selectedDatabase={selectedDatabase}
              selectedSchema={selectedSchema}
              onChange={(schema) => dispatch(setSelectedSchema(schema))}
            />
          </Box>
        )}

        {selectedSchema && schemaDataWithTables && (
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
        />

        {selectedColumns.length === 0 && (
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
