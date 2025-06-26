import React, { useState } from 'react';
import {
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Collapse,
  Box,
  useColorModeValue,
} from '@chakra-ui/react';
import type { Task } from '../tasksSlice';

interface Props {
  tasks: Task[];
}

const TaskTable: React.FC<Props> = ({ tasks }) => {
  const [expanded, setExpanded] = useState<string | null>(null);
  const rowBg = useColorModeValue('white', 'gray.700');
  const expandBg = useColorModeValue('gray.50', 'gray.800');

  return (
    <Table variant="simple" size="sm">
      <Thead>
        <Tr>
          <Th>ID</Th>
          <Th>Статус</Th>
          <Th>Создано</Th>
        </Tr>
      </Thead>
      <Tbody>
        {tasks.map((task) => (
          <React.Fragment key={task.id}>
            <Tr
              bg={rowBg}
              _hover={{ bg: useColorModeValue('gray.100', 'gray.600') }}
              cursor="pointer"
              onClick={() => setExpanded(expanded === task.id ? null : task.id)}
            >
              <Td>{task.id}</Td>
              <Td>{task.status}</Td>
              <Td>{new Date(task.create_date).toLocaleString()}</Td>
            </Tr>
            <Tr>
              <Td colSpan={3} p={0} border="none">
                <Collapse in={expanded === task.id} animateOpacity>
                  <Box p={4} bg={expandBg}>
                    {task.comment || 'Нет комментария'}
                  </Box>
                </Collapse>
              </Td>
            </Tr>
          </React.Fragment>
        ))}
      </Tbody>
    </Table>
  );
};

export default TaskTable;
