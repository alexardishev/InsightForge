import React, { useState, useEffect } from 'react';
import { Box, Heading, VStack, Button } from '@chakra-ui/react';
import { useSelector, useDispatch } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import type { RootState, AppDispatch } from '../../app/store';
import {
  setSelectedDb,
  setSelectedSchema,
  toggleTable,
  toggleColumn,
} from './viewBuilderSlice';
import DatabaseSelector from './components/DatabaseSelector';
import SchemaSelector from './components/SchemaSelector';
import TableSelector from './components/TableSelector';
import ColumnsGrid from './components/ColumnsGrid';
import { useHttp } from '../../hooks/http.hook';
import { appendTables } from '../settings/settingsSlice';

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
  const url = 'http://localhost:8888';

  const [tablesState, setTablesState] = useState<any[]>([]);
  const [page, setPage] = useState(1);
  const pageSize = 20;


  const handleToggleTable = (table: string) => {
    dispatch(toggleTable(table));
  };

  const handleToggleColumn = (table: string, column: string) => {
    dispatch(toggleColumn({ table, column }));
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
      const dbInfo = await request(`${url}/api/get-db`, 'POST', body);
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
    <Box p={8} maxW="1200px" mx="auto">
      <Heading mb={8} textAlign="center">Конструктор витрины</Heading>
      <VStack align="stretch" spacing={6}>
        <Box maxW="md" mx="auto">
          <DatabaseSelector
            data={data}
            selectedDb={selectedDb}
            onChange={(db) => dispatch(setSelectedDb(db))}
          />
        </Box>

        {selectedDb && selectedDatabase && (
          <Box maxW="md" mx="auto">
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
        />
        
        {selectedColumns.length > 0 && (
          <Box textAlign="center">
            <Button onClick={handleBuildView} colorScheme="blue" size="lg">
              Собрать витрину
            </Button>
          </Box>
        )}
      </VStack>
    </Box>
  );
};

export default ViewBuilderPage;
