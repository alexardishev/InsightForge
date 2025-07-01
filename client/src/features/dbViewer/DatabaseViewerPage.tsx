import React, { useState, useEffect } from 'react';
import {
  Box,
  Heading,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Text,
  SimpleGrid,
  useColorModeValue,
  ScaleFade,
  Button,
} from '@chakra-ui/react';
import { useSelector, useDispatch } from 'react-redux';
import {
  useReactTable,
  createColumnHelper,
  getCoreRowModel,
  getSortedRowModel,
  flexRender,
  type SortingState,
} from '@tanstack/react-table';
import type { RootState, AppDispatch } from '../../app/store';
import DatabaseSelector from '../viewBuilder/components/DatabaseSelector';
import SchemaSelector from '../viewBuilder/components/SchemaSelector';
import { setSelectedDb, setSelectedSchema } from '../viewBuilder/viewBuilderSlice';
import { useHttp } from '../../hooks/http.hook';

interface TableRow {
  name: string;
  columns: any[];
}

const columnHelper = createColumnHelper<TableRow>();

const DatabaseViewerPage: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const data = useSelector((state: RootState) => state.settings.dataBaseInfo);
  const connectionsMap = useSelector((state: RootState) => state.settings.connectionsMap);
  const { selectedDb, selectedSchema } = useSelector((state: RootState) => state.viewBuilder);
  const { request } = useHttp();
  const url = 'http://localhost:8888';

  const selectedDatabase = data?.find((db: any) => db.name === selectedDb);
  const selectedSchemaData = selectedDatabase?.schemas?.find((s: any) => s.name === selectedSchema);

  const [tablesState, setTablesState] = useState<TableRow[]>(selectedSchemaData?.tables || []);
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const [sorting, setSorting] = useState<SortingState>([]);
  const [expanded, setExpanded] = useState<string | null>(null);

  const columns = [
    columnHelper.accessor('name', {
      header: 'Таблица',
      cell: info => info.getValue(),
    }),
    columnHelper.accessor(row => row.columns.length, {
      id: 'count',
      header: 'Колонки',
      cell: info => info.getValue(),
    }),
  ];

  const table = useReactTable({
    data: tablesState,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  const rowBg = useColorModeValue('white', 'gray.700');
  const expandBg = useColorModeValue('gray.50', 'gray.800');

  useEffect(() => {
    setTablesState(selectedSchemaData?.tables || []);
    setPage(1);
  }, [selectedSchemaData]);

  const loadMore = async () => {
    const nextPage = page + 1;
    try {
      const body = {
        connection_strings: [{ connection_string: connectionsMap }],
        page: nextPage,
        page_size: pageSize,
      };
      const dbInfo = await request(`${url}/api/get-db`, 'POST', body);
      const db = dbInfo.find((d: any) => d.name === selectedDb);
      const schema = db?.schemas?.find((s: any) => s.name === selectedSchema);
      const newTables = schema?.tables || [];
      if (newTables.length > 0) {
        setTablesState(prev => [...prev, ...newTables]);
        setPage(nextPage);
      }
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <Box p={8} maxW="1000px" mx="auto">
      <Heading mb={8} textAlign="center">Просмотр базы данных</Heading>
      <Box maxW="md" mx="auto" mb={4}>
        <DatabaseSelector data={data} selectedDb={selectedDb} onChange={(db) => dispatch(setSelectedDb(db))} />
      </Box>
      {selectedDb && selectedDatabase && (
        <Box maxW="md" mx="auto" mb={4}>
          <SchemaSelector selectedDatabase={selectedDatabase} selectedSchema={selectedSchema} onChange={(schema) => dispatch(setSelectedSchema(schema))} />
        </Box>
      )}
      {selectedSchema && (
        <Table variant="simple" size="sm">
          <Thead>
            {table.getHeaderGroups().map(headerGroup => (
              <Tr key={headerGroup.id}>
                {headerGroup.headers.map(header => (
                  <Th
                    key={header.id}
                    cursor={header.column.getCanSort() ? 'pointer' : undefined}
                    onClick={header.column.getToggleSortingHandler()}
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </Th>
                ))}
              </Tr>
            ))}
          </Thead>
          <Tbody>
            {table.getRowModel().rows.map(row => (
              <React.Fragment key={row.id}>
                <Tr
                  bg={rowBg}
                  _hover={{ bg: useColorModeValue('gray.100', 'gray.600') }}
                  cursor="pointer"
                  onClick={() => setExpanded(expanded === row.id ? null : row.id)}
                >
                  {row.getVisibleCells().map(cell => (
                    <Td key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </Td>
                  ))}
                </Tr>
                <Tr>
                  <Td colSpan={2} p={0} border="none">
                    <ScaleFade in={expanded === row.id} unmountOnExit>
                      <Box p={4} bg={expandBg}>
                        <Text fontWeight="bold" mb={2}>Колонки:</Text>
                        <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} spacing={2}>
                          {row.original.columns.map(col => (
                            <Box key={col.name} p={2} borderWidth="1px" borderRadius="md">
                              <Text fontSize="sm" fontWeight="medium">{col.name}</Text>
                              <Text fontSize="xs" color="gray.500">{col.type}</Text>
                            </Box>
                          ))}
                        </SimpleGrid>
                      </Box>
                    </ScaleFade>
                  </Td>
                </Tr>
              </React.Fragment>
            ))}
          </Tbody>
        </Table>
        <Box textAlign="center" mt={4}>
          <Button onClick={loadMore}>Загрузить ещё</Button>
        </Box>
      )}
    </Box>
  );
};

export default DatabaseViewerPage;
