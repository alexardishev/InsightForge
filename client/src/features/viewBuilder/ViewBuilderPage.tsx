import React from 'react';
import { Box, Heading, VStack, Text, HStack, Badge, Divider, Stack } from '@chakra-ui/react';
import { useSelector, useDispatch } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import type { RootState, AppDispatch } from '../../app/store';
import {
  setSelectedDatabases,
  setSchemasForDb,
  toggleTable,
  toggleColumn,
  setTableColumns,
  setColumnAlias,
  flattenSelections,
  setTableStatus,
  setSchemaStatus,
} from './viewBuilderSlice';
import DatabaseSelector from './components/DatabaseSelector';
import SchemaSelector from './components/SchemaSelector';
import TableSelector from './components/TableSelector';
import ColumnsGrid from './components/ColumnsGrid';
import { useHttp } from '../../hooks/http.hook';
import { appendTables } from '../settings/settingsSlice';
import FlowLayout from '../../components/FlowLayout';
import { builderSteps } from './flowSteps';

interface DatabaseInfo {
  name: string;
  schemas?: SchemaInfo[];
}

interface SchemaInfo {
  name: string;
  tables?: TableInfo[];
}

interface TableInfo {
  name: string;
  columns?: any[];
  rows?: number;
}

const ViewBuilderPage: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const navigate = useNavigate();
  const data = useSelector((state: RootState) => state.settings.dataBaseInfo) as DatabaseInfo[];
  const selectedConnections = useSelector(
    (state: RootState) => state.settings.selectedConnections,
  );
  const builder = useSelector((state: RootState) => state.viewBuilder);
  const selectedSources = React.useMemo(() => flattenSelections(builder), [builder]);

  const { selectedDatabases, selectedSchemasByDb, selectionsByDbSchema } = builder;

  const { request } = useHttp();
  const url = '/api';
  const pageSize = 20;

  const handleDatabaseChange = (dbs: string[]) => {
    dispatch(setSelectedDatabases(dbs));
  };

  const handleSchemaChange = (db: string, schemas: string[]) => {
    dispatch(setSchemasForDb({ db, schemas }));
  };

  const handleToggleTable = (db: string, schema: string, table: string) => {
    dispatch(toggleTable({ db, schema, table }));
  };

  const handleToggleColumn = (db: string, schema: string, table: string, column: any) => {
    dispatch(
      toggleColumn({
        db,
        schema,
        table,
        column: column.name,
        isPrimaryKey: column.is_primary_key || column.is_pk,
        isUpdateKey: column.is_update_key,
      }),
    );
  };

  const handleSetTableColumns = (db: string, schema: string, table: string, columns: any[]) => {
    dispatch(
      setTableColumns({
        db,
        schema,
        table,
        columns: columns.map((col) => ({
          name: col.name,
          isPrimaryKey: col.is_primary_key || col.is_pk,
          isUpdateKey: col.is_update_key,
        })),
      }),
    );
  };

  const handleAliasChange = (
    db: string,
    schema: string,
    table: string,
    column: string,
    alias: string,
  ) => {
    dispatch(setColumnAlias({ db, schema, table, column, alias }));
  };

  const loadMore = async (db: string, schema: string) => {
    const status = builder.tableStatusByDbSchema[db]?.[schema] || {
      page: 1,
      hasMore: true,
      loading: false,
    };
    if (status.loading || !status.hasMore) return;
    const nextPage = status.page + 1;
    dispatch(setTableStatus({ db, schema, status: { loading: true } }));
    try {
      const body = {
        connection_strings: selectedConnections.map((connection_string) => ({
          connection_string,
        })),
        page: nextPage,
        page_size: pageSize,
      };
      const dbInfo = await request(`${url}/get-db`, 'POST', body);
      const responseDb = dbInfo.find((d: DatabaseInfo) => d.name === db);
      const responseSchema = responseDb?.schemas?.find((s: SchemaInfo) => s.name === schema);
      const newTables = responseSchema?.tables || [];
      if (newTables.length > 0) {
        dispatch(appendTables({ db, schema, tables: newTables }));
      }
      dispatch(
        setTableStatus({
          db,
          schema,
          status: {
            loading: false,
            page: nextPage,
            hasMore: newTables.length >= pageSize,
            error: undefined,
          },
        }),
      );
    } catch (e) {
      console.error(e);
      dispatch(setTableStatus({ db, schema, status: { loading: false, error: 'Ошибка загрузки' } }));
    }
  };

  React.useEffect(() => {
    selectedDatabases.forEach((dbName) => {
      const hasSchemas =
        data?.find((db) => db.name === dbName)?.schemas &&
        (data.find((db) => db.name === dbName)?.schemas?.length ?? 0) > 0;
      if (hasSchemas) {
        dispatch(setSchemaStatus({ db: dbName, status: { loaded: true, loading: false } }));
      }
    });
  }, [data, dispatch, selectedDatabases]);

  const selectedSchemaViews = React.useMemo(
    () =>
      selectedDatabases.flatMap((db) => {
        const dbData = data?.find((item) => item.name === db);
        const schemaNames = selectedSchemasByDb[db] || [];
        return schemaNames.map((schema) => {
          const schemaData = dbData?.schemas?.find((s) => s.name === schema);
          const selection = selectionsByDbSchema[db]?.[schema];
          return {
            db,
            schema,
            schemaData,
            selectedTables: selection?.selectedTables || [],
            selectedColumns: selection?.selectedColumns || [],
            tableStatus: builder.tableStatusByDbSchema[db]?.[schema],
          };
        });
      }),
    [builder.tableStatusByDbSchema, data, selectedDatabases, selectedSchemasByDb, selectionsByDbSchema],
  );

  const isNextDisabled = selectedSources.every((source) => source.selectedColumns.length === 0);

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
      isNextDisabled={isNextDisabled}
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
            data={data || []}
            selectedDbs={selectedDatabases}
            onChange={handleDatabaseChange}
          />
        </Box>

        <Stack spacing={6}>
          {selectedDatabases.map((db) => {
            const dbData = data?.find((item) => item.name === db);
            const schemas = dbData?.schemas || [];
            const selectedSchemas = selectedSchemasByDb[db] || [];
            return (
              <Box key={db} borderWidth="1px" borderRadius="lg" p={4}>
                <VStack align="stretch" spacing={4}>
                  <HStack justify="space-between">
                    <Heading size="md">{db}</Heading>
                    <Badge colorScheme="cyan">{selectedSchemas.length} схем</Badge>
                  </HStack>
                  <SchemaSelector
                    database={db}
                    schemas={schemas}
                    selectedSchemas={selectedSchemas}
                    onChange={(schemasValue) => handleSchemaChange(db, schemasValue)}
                  />

                  {selectedSchemas.map((schema) => {
                    const schemaData = schemas.find((s) => s.name === schema);
                    const selection = selectionsByDbSchema[db]?.[schema];
                    const tableStatus = builder.tableStatusByDbSchema[db]?.[schema];
                    return (
                      <Box key={`${db}-${schema}`} borderWidth="1px" borderRadius="lg" p={3}>
                        <TableSelector
                          selectedSchemaData={schemaData}
                          selectedTables={selection?.selectedTables || []}
                          onToggleTable={(table) => handleToggleTable(db, schema, table)}
                          onLoadMore={() => loadMore(db, schema)}
                          hasMore={tableStatus?.hasMore ?? true}
                          isLoading={tableStatus?.loading}
                          dbLabel={db}
                          schemaLabel={schema}
                        />
                      </Box>
                    );
                  })}
                </VStack>
              </Box>
            );
          })}
        </Stack>

        <ColumnsGrid
          sources={selectedSchemaViews}
          onToggleColumn={handleToggleColumn}
          onSetTableColumns={handleSetTableColumns}
          onAliasChange={handleAliasChange}
        />

        {isNextDisabled && (
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
