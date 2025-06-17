import React from 'react';
import { Box, Heading, VStack, Button } from '@chakra-ui/react';
import { useSelector, useDispatch } from 'react-redux';
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

interface Column {
  name: string;
  type?: string;
  is_nullable?: boolean;
  is_primary_key?: boolean;
  is_pk?: boolean;
  is_fk?: boolean;
  default?: string;
  is_unique?: boolean;
  description?: string;
}

interface Table {
  name: string;
  columns: Column[];
}

interface Schema {
  name: string;
  tables: Table[];
}

interface Source {
  name: string;
  schemas: Schema[];
}

interface View {
  view_name: string;
  sources: Source[];
  joins: any[];
}

interface SelectedColumn {
  table: string;
  column: string;
}

const ViewBuilderPage: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
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
    const source: Source = {
      name: selectedDb,
      schemas: [
        {
          name: selectedSchema,
          tables: selectedTables.map((tableName) => {
            const tableData = selectedSchemaData.tables.find((t: any) => t.name === tableName);
            return {
              name: tableName,
              columns: tableData.columns
                .filter((col: any) =>
                  selectedColumns.some((c) => c.table === tableName && c.column === col.name)
                )
                .map((col: any) => ({
                  name: col.name,
                  type: col.type,
                  is_nullable: col.is_nullable,
                  is_primary_key: col.is_primary_key || col.is_pk,
                  is_fk: col.is_fk,
                  default: col.default,
                  is_unq: col.is_unique,
                })),
            };
          }),
        },
      ],
    };

    const view: View = {
      view_name: 'MyView',
      sources: [source],
      joins: [],
    };

    console.log('Собранная витрина:', view);
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
