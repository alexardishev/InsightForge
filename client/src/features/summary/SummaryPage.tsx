import React from 'react';
import { Box, Heading } from '@chakra-ui/react';
import { useSelector } from 'react-redux';
import type { RootState } from '../../app/store';
import ViewPreview from './components/ViewPreview';
import SummaryActions from './components/SummaryActions';

const SummaryPage: React.FC = () => {
  const settings = useSelector((state: RootState) => state.settings);
  const builder = useSelector((state: RootState) => state.viewBuilder);

  const data = settings.dataBaseInfo;

  const selectedDatabase = data?.find((db: any) => db.name === builder.selectedDb);
  const selectedSchemaData = selectedDatabase?.schemas?.find((s: any) => s.name === builder.selectedSchema);

  if (!selectedSchemaData) return null;

  const source = {
    name: builder.selectedDb,
    schemas: [
      {
        name: builder.selectedSchema,
        tables: builder.selectedTables.map((tableName) => {
          const tableData = selectedSchemaData.tables.find((t: any) => t.name === tableName);
          return {
            name: tableName,
            columns: tableData.columns
              .filter((col: any) => builder.selectedColumns.some((c) => c.table === tableName && c.column === col.name))
              .map((col: any) => {
                const key = `${tableName}.${col.name}`;
                const selectedColumn = builder.selectedColumns.find(
                  (c) => c.table === tableName && c.column === col.name,
                );
                const base = {
                  name: col.name,
                  type: col.type,
                  is_nullable: col.is_nullable,
                  is_primary_key: col.is_primary_key || col.is_pk,
                  is_fk: col.is_fk,
                  default: col.default,
                  is_unq: col.is_unique,
                  view_key:
                    selectedColumn?.viewKey || col.view_key,
                  is_update_key:
                    selectedColumn?.isUpdateKey ??
                    col.is_update_key ??
                    col.is_primary_key ||
                    col.is_pk ||
                    false,
                };
                const tr = builder.transformations[key];
                return tr ? { ...base, transform: tr } : base;
              }),
          };
        }),
      },
    ],
  };

  const view = {
    view_name: builder.viewName,
    sources: [source],
    joins: builder.joins,
  };

  return (
    <Box p={8} maxW="900px" mx="auto">
      <Heading mb={4} textAlign="center">
        Итоговая схема
      </Heading>
      <ViewPreview view={view} />
      <SummaryActions view={view} />
    </Box>
  );
};

export default SummaryPage;
