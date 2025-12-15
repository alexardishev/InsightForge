import React, { useState } from 'react';
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
import { setTransformation, setUpdateKey, setViewKey } from './viewBuilderSlice';
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
  const {
    selectedDb,
    selectedSchema,
    selectedTables,
    selectedColumns,
    transformations,
  } = useSelector((state: RootState) => state.viewBuilder);

  const selectedDatabase = data?.find((db: any) => db.name === selectedDb);
  const selectedSchemaData = selectedDatabase?.schemas?.find((s: any) => s.name === selectedSchema);

  const initial: Record<string, LocalTransformState> = {};
  selectedColumns.forEach((c) => {
    const key = `${c.table}.${c.column}`;
    const existing = transformations[key];
    initial[key] = {
      type: existing?.type || '',
      output: existing?.output_column || '',
      mapping: existing?.mapping?.mapping ? JSON.stringify(existing.mapping.mapping, null, 2) : '',
      mappingJson: existing?.mapping?.mapping_json ? JSON.stringify(existing.mapping.mapping_json, null, 2) : '',
    };
  });

  const [local, setLocal] = useState<Record<string, LocalTransformState>>(initial);

  const updateField = (key: string, field: keyof LocalTransformState, value: string) => {
    setLocal((prev) => ({ ...prev, [key]: { ...prev[key], [field]: value } }));
  };

  const handleSave = async () => {
    Object.entries(local).forEach(([key, val]) => {
      const [table, column] = key.split('.');
      if (!val.type) {
        dispatch(setTransformation({ table, column, transform: null }));
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
        output_column: val.output || column,
        mapping,
      };
      dispatch(setTransformation({ table, column, transform }));
    });

    navigate('/summary');
  };

  const renderColumn = (table: string, column: any) => {
    const key = `${table}.${column.name}`;
    if (!local[key]) return null;
    const val = local[key];
    const otherColumns = selectedColumns.filter(
      (c) => c.table !== table || c.column !== column.name,
    );
    const selectedCol = selectedColumns.find(
      (c) => c.table === table && c.column === column.name,
    );
    return (
      <Box
        key={key}
        p={4}
        borderWidth="1px"
        borderRadius="md"
        mb={4}
        background={useColorModeValue('gray.50', 'gray.700')}
      >
        <Text mb={2} fontWeight="bold">{table}.{column.name}</Text>
        <Select
          mb={2}
          placeholder="view_key"
          value={selectedCol?.viewKey || ''}
          onChange={(e) =>
            dispatch(
              setViewKey({
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

        {selectedTables.map((tableName) => {
          const tableData = selectedSchemaData?.tables.find((t: any) => t.name === tableName);
          return (
            <Box key={tableName}>
              <HStack justify="space-between" mb={2}>
                <Text fontSize="lg" fontWeight="bold">{tableName}</Text>
                <Tag colorScheme="cyan">{tableData?.columns?.length || 0} колонок</Tag>
              </HStack>
              <Divider mb={2} />
              {tableData?.columns
                .filter((col: any) => selectedColumns.some((c) => c.table === tableName && c.column === col.name))
                .map((col: any) => renderColumn(tableName, col))}
            </Box>
          );
        })}
      </VStack>
    </FlowLayout>
  );
};

export default TransformBuilderPage;
