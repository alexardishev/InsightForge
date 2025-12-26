import React from 'react';
import { Box, Heading, Text, VStack, Divider, Badge, HStack } from '@chakra-ui/react';
import { useSelector } from 'react-redux';
import type { RootState } from '../../app/store';
import ViewPreview from './components/ViewPreview';
import SummaryActions from './components/SummaryActions';
import FlowLayout from '../../components/FlowLayout';
import { builderSteps } from '../viewBuilder/flowSteps';

const SummaryPage: React.FC = () => {
  const settings = useSelector((state: RootState) => state.settings);
  const builder = useSelector((state: RootState) => state.viewBuilder);

  const data = settings.dataBaseInfo;

  const sources = builder.selectedSources
    .map((source) => {
      const selectedDatabase = data?.find((db: any) => db.name === source.db);
      const selectedSchemaData = selectedDatabase?.schemas?.find(
        (s: any) => s.name === source.schema,
      );
      if (!selectedSchemaData) return null;
      return {
        name: source.db,
        schemas: [
          {
            name: source.schema,
            tables: source.selectedTables.map((tableName) => {
              const tableData = selectedSchemaData.tables.find(
                (t: any) => t.name === tableName,
              );
              return {
                name: tableName,
                columns:
                  tableData?.columns
                    .filter((col: any) =>
                      source.selectedColumns.some(
                        (c) => c.table === tableName && c.column === col.name,
                      ),
                    )
                    .map((col: any) => {
                      const key = `${source.db}.${source.schema}.${tableName}.${col.name}`;
                      const selectedColumn = source.selectedColumns.find(
                        (c) => c.table === tableName && c.column === col.name,
                      );
                      const base = {
                        name: col.name,
                        alias: selectedColumn?.alias,
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
                          (col.is_primary_key || col.is_pk || false),
                      };
                      const tr = builder.transformations[key];
                      return tr ? { ...base, transform: tr } : base;
                    }) || [],
              };
            }),
          },
        ],
      };
    })
    .filter((source): source is NonNullable<typeof source> => Boolean(source));

  if (sources.length === 0) return null;

  const view = {
    view_name: builder.viewName,
    sources,
    joins: builder.joins,
  };

  return (
    <FlowLayout
      steps={builderSteps}
      currentStep={4}
      onBack={() => window.history.back()}
      primaryLabel="Запустить"
      onNext={() => null}
    >
      <VStack align="stretch" spacing={4}>
        <Box>
          <Heading size="lg">Review & Apply</Heading>
          <Text color="text.muted" mt={2}>
            Проверь выбранные источники, связи и трансформации перед запуском. Фокус на прозрачных
            предупреждениях и безопасности.
          </Text>
          <HStack spacing={3} mt={3} flexWrap="wrap">
            {builder.selectedSources.map((source) => (
              <Badge key={`${source.db}.${source.schema}`} colorScheme="cyan">
                {source.db}.{source.schema}
              </Badge>
            ))}
          </HStack>
        </Box>
        <Divider borderColor="border.subtle" />
        <ViewPreview view={view} />
        <SummaryActions view={view} />
      </VStack>
    </FlowLayout>
  );
};

export default SummaryPage;
