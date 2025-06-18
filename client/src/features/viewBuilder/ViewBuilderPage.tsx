import React from 'react';
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

const ViewBuilderPage: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const navigate = useNavigate();
  const data = useSelector((state: RootState) => state.settings.dataBaseInfo);
  const { selectedDb, selectedSchema, selectedTables, selectedColumns } =
    useSelector((state: RootState) => state.viewBuilder);


  const handleToggleTable = (table: string) => {
    dispatch(toggleTable(table));
  };

  const handleToggleColumn = (table: string, column: string) => {
    dispatch(toggleColumn({ table, column }));
  };

  const selectedDatabase = data?.find((db: any) => db.name === selectedDb);
  const selectedSchemaData = selectedDatabase?.schemas?.find((schema: any) => schema.name === selectedSchema);

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

        {selectedSchema && selectedSchemaData && (
          <TableSelector
            selectedSchemaData={selectedSchemaData}
            selectedTables={selectedTables}
            onToggleTable={handleToggleTable}
          />
        )}

        <ColumnsGrid
          selectedTables={selectedTables}
          selectedSchemaData={selectedSchemaData}
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
