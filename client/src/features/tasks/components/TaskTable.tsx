import React, { useState } from 'react';
import {
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Box,
  useColorModeValue,
  Badge,
  ScaleFade,
  Progress,
  Text,
  HStack,
} from '@chakra-ui/react';
import type { Task } from '../tasksSlice';

interface Props {
  tasks: Task[];
}

const TaskTable: React.FC<Props> = ({ tasks }) => {
  const [expanded, setExpanded] = useState<string | null>(null);
  const rowBg = useColorModeValue('rgba(255,255,255,0.04)', 'rgba(255,255,255,0.04)');
  const expandBg = useColorModeValue('rgba(255,255,255,0.06)', 'rgba(255,255,255,0.08)');

  const getStatusColor = (status: string) => {
    const s = status.toLowerCase();
    if (s.includes('success')) return 'green';
    if (s.includes('fail') || s.includes('error')) return 'red';
    return 'yellow';
  };

  return (
    <Table variant="dataGrid" size="sm" width="100%">
      <Thead>
        <Tr>
          <Th>ID</Th>
          <Th>Статус</Th>
          <Th>Создано</Th>
          <Th>Прогресс</Th>
        </Tr>
      </Thead>
      <Tbody>
        {tasks.map((task) => (
          <React.Fragment key={task.id}>
            <Tr
              bg={rowBg}
              _hover={{ bg: useColorModeValue('gray.100', 'gray.700') }}
              cursor="pointer"
              transition="transform 0.1s"
              _active={{ transform: 'scale(0.98)' }}
              onClick={() => setExpanded(expanded === task.id ? null : task.id)}
            >
              <Td>{task.id}</Td>
              <Td>
                <Badge colorScheme={getStatusColor(task.status)}>{task.status}</Badge>
              </Td>
              <Td>{new Date(task.create_date).toLocaleString()}</Td>
              <Td>
                <Progress value={task.progress ?? 0} size="xs" colorScheme={getStatusColor(task.status)} borderRadius="full" />
              </Td>
            </Tr>
            <Tr>
              <Td colSpan={3} p={0} border="none">
                <ScaleFade in={expanded === task.id} unmountOnExit>
                  <Box p={4} bg={expandBg} borderRadius="md">
                    <HStack justify="space-between" mb={2}>
                      <Text fontWeight="semibold">Комментарий</Text>
                      <Badge variant="outline">{task.run_id || '—'}</Badge>
                    </HStack>
                    <Text color="text.muted">{task.comment || 'Нет комментария'}</Text>
                  </Box>
                </ScaleFade>
              </Td>
            </Tr>
          </React.Fragment>
        ))}
      </Tbody>
    </Table>
  );
};

export default TaskTable;
