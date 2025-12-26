import React, { useState, useMemo } from 'react';
import {
  Box,
  Heading,
  VStack,
  Select,
  Input,
  Textarea,
  Text,
  Divider,
  useColorModeValue,
  Checkbox,
  Tooltip,
  HStack,
  Alert,
  AlertIcon,
  Tag,
} from '@chakra-ui/react';
import { useSelector, useDispatch } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import type { RootState, AppDispatch } from '../../app/store';
import { setTransformation, setUpdateKey, setViewKey, flattenSelections } from './viewBuilderSlice';
import { InfoIcon } from '@chakra-ui/icons';
import FlowLayout from '../../components/FlowLayout';
import { builderSteps } from './flowSteps';

interface LocalTransformState {
  type: string;
  output: string;
  mapping: string;
  mappingJson: string;
}

const TransformBuilderPage: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const navigate = useNavigate();

  const data = useSelector((state: RootState) => state.settings.dataBaseInfo);
  const builder = useSelector((state: RootState) => state.viewBuilder);
  const selectedSources = useMemo(() => flattenSelections(builder), [builder]);
  const { transformations } = builder;

  const [local, setLocal] = useState<Record<string, LocalTransformState>>({});

  const buildKey = (payload: { db: string; schema: string; table: string; column: string }) =>
    `${payload.db}.${payload.schema}.${payload.table}.${payload.column}`;

  const selectedColumns = selectedSources.flatMap((source) => source.selectedColumns);

  const rebuildLocal = () => {
    const next: Record<string, LocalTransformState> = {};
    selectedColumns.forEach((c) => {
      const key = buildKey(c);
      const existing = transformations[key];
      next[key] = {
        type: existing?.type || '',
        output: existing?.output_column || c.alias || '',
        mapping: existing?.mapping?.mapping ? JSON.stringify(existing.mapping.mapping, null, 2) : '',
        mappingJson: existing?.mapping?.mapping_json
          ? JSON.stringify(existing.mapping.mapping_json, null, 2)
          : '',
      };
    });
    setLocal(next);
  };

  React.useEffect(() => {
    rebuildLocal();
  }, [selectedColumns, transformations]);

  const updateField = (key: string, field: keyof LocalTransformState, value: string) => {
    setLocal((prev) => ({ ...prev, [key]: { ...prev[key], [field]: value } }));
  };

  const handleSave = async () => {
    Object.entries(local).forEach(([key, val]) => {
      const [db, schema, table, column] = key.split('.');
      const selectedCol = selectedColumns.find(
        (c) => c.table === table && c.column === column && c.db === db && c.schema === schema,
      );
      if (!val.type) {
        dispatch(
          setTransformation({ db, schema, table, column, transform: null }),
        );
        return;
      }
      let mapping: any = {};
      if (val.type === 'FieldTransform' && val.mapping) {
        try {
          mapping = { type_map: 'FieldTransform', alias_new_column_transform: val.output, mapping: JSON.parse(val.mapping) };
        } catch {}
      }
      if (val.type === 'JSON' && val.mappingJson) {
        try {
          mapping = { type_map: 'JSON', mapping_json: JSON.parse(val.mappingJson) };
        } catch {}
      }
      const transform = {
        type: val.type,
        mode: 'Mapping',
        output_column: val.output || selectedCol?.alias || column,
        mapping,
      };
      dispatch(setTransformation({ db, schema, table, column, transform }));
    });

    navigate('/summary');
  };

  const cardBackground = useColorModeValue('gray.50', 'gray.700');

  const renderColumn = (db: string, schema: string, table: string, column: any) => {
    const key = `${db}.${schema}.${table}.${column.name}`;
    if (!local[key]) return null;
    const val = local[key];
    const otherColumns = selectedColumns.filter(
      (c) =>
        c.table !== table || c.column !== column.name || c.db !== db || c.schema !== schema,
    );
    const selectedCol = selectedColumns.find(
      (c) =>
        c.table === table &&
        c.column === column.name &&
        c.db === db &&
        c.schema === schema,
    );
    return (
      <Box
        key={key}
        p={4}
        borderWidth="1px"
        borderRadius="md"
        mb={4}
        background={cardBackground}
      >
        <Text mb={2} fontWeight="bold">{table}.{column.name}</Text>
        <Select
          mb={2}
          placeholder="view_key"
          value={selectedCol?.viewKey || ''}
          onChange={(e) =>
            dispatch(
              setViewKey({
                db,
                schema,
                table,
                column: column.name,
                viewKey: e.target.value,
              }),
            )
          }
        >
          <option value="">Без view_key</option>
          {otherColumns.map((c) => (
            <option key={`${c.table}.${c.column}`} value={c.column}>
              {c.table}.{c.column}
            </option>
          ))}
        </Select>
        <HStack alignItems="center" mb={2} spacing={2}>
          <Checkbox
            isChecked={selectedCol?.isUpdateKey || false}
            onChange={(e) =>
              dispatch(
                setUpdateKey({
                  db,
                  schema,
                  table,
                  column: column.name,
                  isUpdateKey: e.target.checked,
                }),
              )
            }
          >
            is_update_key
          </Checkbox>
          <Tooltip label="Отмеченные колонки участвуют в логике обновления данных" hasArrow>
            <InfoIcon color="gray.500" />
          </Tooltip>
        </HStack>
        <Select mb={2} value={val.type} onChange={(e) => updateField(key, 'type', e.target.value)}>
          <option value="">Без трансформации</option>
          <option value="FieldTransform">FieldTransform</option>
          <option value="JSON">JSON</option>
        </Select>
        <Input
          mb={2}
          placeholder="Output column"
          value={val.output}
          onChange={(e) => updateField(key, 'output', e.target.value)}
        />
        {val.type === 'FieldTransform' && (
          <Textarea mb={2} placeholder='{"1":"A"}' value={val.mapping} onChange={(e) => updateField(key, 'mapping', e.target.value)} />
        )}
        {val.type === 'JSON' && (
          <Textarea mb={2} placeholder='[{"type_field":"int","mapping":{"json_field":"out_col"}}]' value={val.mappingJson} onChange={(e) => updateField(key, 'mappingJson', e.target.value)} />
        )}
      </Box>
    );
  };

  return (
    <FlowLayout
      steps={builderSteps}
      currentStep={3}
      onBack={() => navigate('/joins')}
      onNext={handleSave}
      primaryLabel="К обзору"
      secondaryLabel="Назад к джоинам"
    >
      <VStack align="stretch" spacing={4}>
        <Box>
          <Heading size="lg">Transform Builder</Heading>
          <Text color="text.muted" mt={2}>
            Настрой переименования, JSON-раскладку и surrogate/view ключи. Мы подсветим конфликты
            алиасов.
          </Text>
        </Box>

        <Alert status="warning" borderRadius="md" variant="left-accent">
          <AlertIcon />
          Перед применением проверь конфликты алиасов и обновляемые поля.
        </Alert>

        {selectedSources.map((source) => {
          const dbData = data?.find((db: any) => db.name === source.db);
          const schemaData = dbData?.schemas?.find((s: any) => s.name === source.schema);
          return source.selectedTables.map((tableName) => {
            const tableData = schemaData?.tables.find((t: any) => t.name === tableName);
            const key = `${source.db}.${source.schema}.${tableName}`;
            return (
              <Box key={key}>
                <HStack justify="space-between" mb={2}>
                  <Text fontSize="lg" fontWeight="bold">
                    {source.db}.{source.schema}.{tableName}
                  </Text>
                  <Tag colorScheme="cyan">{tableData?.columns?.length || 0} колонок</Tag>
                </HStack>
                <Divider mb={2} />
                {tableData?.columns
                  .filter((col: any) =>
                    selectedColumns.some(
                      (c) =>
                        c.table === tableName &&
                        c.column === col.name &&
                        c.db === source.db &&
                        c.schema === source.schema,
                    ),
                  )
                  .map((col: any) => renderColumn(source.db, source.schema, tableName, col))}
              </Box>
            );
          });
        })}
      </VStack>
    </FlowLayout>
  );
};

export default TransformBuilderPage;
