import React, { useState } from 'react';
import {
  Box,
  Heading,
  VStack,
  Select,
  Input,
  Textarea,
  Button,
  Text,
  Divider,
} from '@chakra-ui/react';
import { useSelector, useDispatch } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import type { RootState, AppDispatch } from '../../app/store';
import { setTransformation } from './viewBuilderSlice';
import { useHttp } from '../../hooks/http.hook';

interface LocalTransformState {
  type: string;
  output: string;
  mapping: string;
  mappingJson: string;
}

const TransformBuilderPage: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const navigate = useNavigate();
  const { request } = useHttp();

  const data = useSelector((state: RootState) => state.settings.dataBaseInfo);
  const {
    selectedDb,
    selectedSchema,
    selectedTables,
    selectedColumns,
    joins,
    viewName,
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

    if (!selectedSchemaData) return;
    const source = {
      name: selectedDb,
      schemas: [
        {
          name: selectedSchema,
          tables: selectedTables.map((tableName) => {
            const tableData = selectedSchemaData.tables.find((t: any) => t.name === tableName);
            return {
              name: tableName,
              columns: tableData.columns
                .filter((col: any) => selectedColumns.some((c) => c.table === tableName && c.column === col.name))
                .map((col: any) => {
                  const key = `${tableName}.${col.name}`;
                  const base = {
                    name: col.name,
                    type: col.type,
                    is_nullable: col.is_nullable,
                    is_primary_key: col.is_primary_key || col.is_pk,
                    is_fk: col.is_fk,
                    default: col.default,
                    is_unq: col.is_unique,
                    view_key: col.view_key,
                    is_update_key: col.is_update_key,
                  };
                  const tr = transformations[key];
                  return tr ? { ...base, transform: tr } : base;
                }),
            };
          }),
        },
      ],
    };

    const view = {
      view_name: viewName,
      sources: [source],
      joins,
    };

    try {
      await request('http://localhost:8888/api/upload-schem', 'POST', view);
      navigate('/settings');
    } catch (e) {
      console.error(e);
    }
  };

  const renderColumn = (table: string, column: any) => {
    const key = `${table}.${column.name}`;
    if (!local[key]) return null;
    const val = local[key];
    return (
      <Box key={key} p={4} borderWidth="1px" borderRadius="md" mb={4}
           background="gray.700">
        <Text mb={2} fontWeight="bold">{table}.{column.name}</Text>
        <Select mb={2} value={val.type} onChange={(e) => updateField(key, 'type', e.target.value)}>
          <option value="">Без трансформации</option>
          <option value="FieldTransform">FieldTransform</option>
          <option value="JSON">JSON</option>
        </Select>
        <Input mb={2} placeholder="Output column" value={val.output} onChange={(e) => updateField(key, 'output', e.target.value)} />
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
    <Box p={8} maxW="900px" mx="auto">
      <Heading mb={8} textAlign="center">Трансформации</Heading>
      <VStack align="stretch" spacing={4}>
        {selectedTables.map((tableName) => {
          const tableData = selectedSchemaData?.tables.find((t: any) => t.name === tableName);
          return (
            <Box key={tableName}>
              <Text fontSize="lg" fontWeight="bold" mb={2}>{tableName}</Text>
              <Divider mb={2} />
              {tableData?.columns
                .filter((col: any) => selectedColumns.some((c) => c.table === tableName && c.column === col.name))
                .map((col: any) => renderColumn(tableName, col))}
            </Box>
          );
        })}
        <Button colorScheme="blue" onClick={handleSave} alignSelf="center">
          Создать витрину
        </Button>
      </VStack>
    </Box>
  );
};

export default TransformBuilderPage;
