import React, { useEffect } from 'react';
import { Box, Heading, Card, CardBody, Text } from '@chakra-ui/react';
import { useDispatch, useSelector } from 'react-redux';
import type { AppDispatch, RootState } from '../../app/store';
import { useHttp } from '../../hooks/http.hook';
import TaskTable from './components/TaskTable';
import PaginationControls from './components/PaginationControls';
import { setTasks, setPage, setPageSize } from './tasksSlice';

const TasksPage: React.FC = () => {
  const dispatch = useDispatch<AppDispatch>();
  const { tasks, page, pageSize } = useSelector((state: RootState) => state.tasks);
  const { request } = useHttp();

  const fetchTasks = async () => {
    try {
      const data = await request('/api/get-tasks', 'POST', {
        page,
        page_size: pageSize,
      });
      dispatch(setTasks(data));
    } catch (e) {
      console.error(e);
    }
  };

  useEffect(() => {
    fetchTasks();
  }, [page, pageSize]);

  return (
    <Box>
      <Heading mb={3}>Мониторинг задач</Heading>
      <Text color="text.muted" mb={4}>
        Статусы, прогресс и комментарии по всем запускам. Кликни по строке, чтобы раскрыть детали.
      </Text>
      <Card variant="surface">
        <CardBody>
          <TaskTable tasks={tasks} />
          <PaginationControls
            page={page}
            pageSize={pageSize}
            onPageChange={(p) => dispatch(setPage(p))}
            onPageSizeChange={(s) => dispatch(setPageSize(s))}
          />
        </CardBody>
      </Card>
    </Box>
  );
};

export default TasksPage;
