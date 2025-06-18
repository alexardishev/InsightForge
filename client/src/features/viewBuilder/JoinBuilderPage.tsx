import React, { useState } from 'react';
import {
  Box,
  Heading,
  VStack,
  HStack,
  Select,
  Button,
  Text,
  Input,
  List,
  ListItem,
  IconButton,
} from '@chakra-ui/react';
import { useDispatch, useSelector } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import type { RootState, AppDispatch } from '../../app/store';
import { addJoin, removeJoin, setViewName } from './viewBuilderSlice';
import { DeleteIcon } from '@chakra-ui/icons';

interface Column {
  name: string;
}

const JoinBuilderPage: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const navigate = useNavigate();

  const data = useSelector((state: RootState) => state.settings.dataBaseInfo);
  const {
    selectedDb,
    selectedSchema,
    selectedTables,
    selectedColumns,
    joins,
    viewName,
  } = useSelector((state: RootState) => state.viewBuilder);

  const selectedDatabase = data?.find((db: any) => db.name === selectedDb);
  const selectedSchemaData = selectedDatabase?.schemas?.find((s: any) => s.name === selectedSchema);

  const [mainTable, setMainTable] = useState('');
  const [joinTable, setJoinTable] = useState('');
  const [mainColumn, setMainColumn] = useState('');
  const [joinColumn, setJoinColumn] = useState('');

  const handleAddJoin = () => {
    if (!mainTable || !joinTable || !mainColumn || !joinColumn) return;
    const join = {
      inner: {
        source: selectedDb,
        schema: selectedSchema,
        table: joinTable,
        main_table: mainTable,
        column_first: mainColumn,
        column_second: joinColumn,
      },
    };
    dispatch(addJoin(join));
    setJoinTable('');
    setMainColumn('');
    setJoinColumn('');
  };

  const handleNext = () => {
    navigate('/transforms');
  };

  const getColumnsForTable = (tableName: string): Column[] => {
    const table = selectedSchemaData?.tables.find((t: any) => t.name === tableName);
    return table ? table.columns : [];
  };

  return (
    <Box p={8} maxW="800px" mx="auto">
      <Heading mb={8} textAlign="center">
        Настройка джоинов
      </Heading>
      <VStack spacing={4} align="stretch">
        <Box>
          {selectedTables.map((table) => (
            <Box key={table} mb={2}>
              <Text fontWeight="bold">{table}</Text>
              <List pl={4} styleType="disc">
                {selectedColumns
                  .filter((c) => c.table === table)
                  .map((c) => (
                    <ListItem key={c.column}>{c.column}</ListItem>
                  ))}
              </List>
            </Box>
          ))}
        </Box>
        <Input
          placeholder="Имя витрины"
          value={viewName}
          onChange={(e) => dispatch(setViewName(e.target.value))}
        />
        <HStack>
          <Select placeholder="Основная таблица" value={mainTable} onChange={(e) => setMainTable(e.target.value)}>
            {selectedTables.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </Select>
          <Select placeholder="Колонка" value={mainColumn} onChange={(e) => setMainColumn(e.target.value)}>
            {getColumnsForTable(mainTable).map((col) => (
              <option key={col.name} value={col.name}>
                {col.name}
              </option>
            ))}
          </Select>
        </HStack>
        <HStack>
          <Select placeholder="Таблица для join" value={joinTable} onChange={(e) => setJoinTable(e.target.value)}>
            {selectedTables.map((t) => (
              <option key={t} value={t}>
                {t}
              </option>
            ))}
          </Select>
          <Select placeholder="Колонка" value={joinColumn} onChange={(e) => setJoinColumn(e.target.value)}>
            {getColumnsForTable(joinTable).map((col) => (
              <option key={col.name} value={col.name}>
                {col.name}
              </option>
            ))}
          </Select>
        </HStack>
        <Button onClick={handleAddJoin} colorScheme="teal" alignSelf="flex-start">
          Добавить join
        </Button>

        <List spacing={3} w="100%">
          {joins.map((j, idx) => (
            <ListItem key={idx} display="flex" alignItems="center" justifyContent="space-between">
              <Text>{`${j.inner.main_table}.${j.inner.column_first} = ${j.inner.table}.${j.inner.column_second}`}</Text>
              <IconButton
                aria-label="delete"
                icon={<DeleteIcon />}
                size="sm"
                onClick={() => dispatch(removeJoin(idx))}
              />
            </ListItem>
          ))}
        </List>

        <Button colorScheme="blue" onClick={handleNext} alignSelf="center">
          Далее
        </Button>
      </VStack>
    </Box>
  );
};

export default JoinBuilderPage;
