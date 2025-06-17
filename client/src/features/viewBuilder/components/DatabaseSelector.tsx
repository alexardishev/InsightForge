import React from 'react';
import { Select } from '@chakra-ui/react';

interface Props {
  data: any[];
  selectedDb: string;
  onChange: (db: string) => void;
}

const DatabaseSelector: React.FC<Props> = ({ data, selectedDb, onChange }) => (
  <Select
    placeholder={data?.length > 0 ? 'Выберите базу данных' : 'Нет доступных баз данных'}
    value={selectedDb}
    onChange={(e) => onChange(e.target.value)}
  >
    {data?.map((db: any, index: number) => (
      <option key={index} value={db?.name}>
        {db?.name}
      </option>
    ))}
  </Select>
);

export default DatabaseSelector;
