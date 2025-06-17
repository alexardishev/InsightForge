import React from 'react';
import { Select } from '@chakra-ui/react';

interface Props {
  selectedDatabase: any;
  selectedSchema: string;
  onChange: (schema: string) => void;
}

const SchemaSelector: React.FC<Props> = ({ selectedDatabase, selectedSchema, onChange }) => {
  if (!selectedDatabase) return null;

  return (
    <Select
      placeholder="Выберите схему"
      value={selectedSchema}
      onChange={(e) => onChange(e.target.value)}
    >
      {selectedDatabase.schemas?.map((schema: any, index: number) => (
        <option key={index} value={schema?.name}>
          {schema?.name}
        </option>
      ))}
    </Select>
  );
};

export default SchemaSelector;
