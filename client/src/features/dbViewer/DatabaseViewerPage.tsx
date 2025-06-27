import React, { useState } from 'react';
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

interface TableRow {
  name: string;
  columns: any[];
}

const columnHelper = createColumnHelper<TableRow>();

const DatabaseViewerPage: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const data = useSelector((state: RootState) => state.settings.dataBaseInfo);
  const { selectedDb, selectedSchema } = useSelector((state: RootState) => state.viewBuilder);

  const selectedDatabase = data?.find((db: any) => db.name === selectedDb);
  const selectedSchemaData = selectedDatabase?.schemas?.find((s: any) => s.name === selectedSchema);

  const tables: TableRow[] = selectedSchemaData?.tables || [];

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
    data: tables,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });

  const rowBg = useColorModeValue('white', 'gray.700');
  const expandBg = useColorModeValue('gray.50', 'gray.800');

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
      )}
    </Box>
  );
};

export default DatabaseViewerPage;
