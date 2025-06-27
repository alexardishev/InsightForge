import React, { useEffect } from 'react';
import { Box, Heading } from '@chakra-ui/react';
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
      const data = await request('http://localhost:8888/api/get-tasks', 'POST', {
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
    <Box p={8} maxW="1200px" mx="auto">
      <Heading mb={4} textAlign="center">
        Задачи
      </Heading>
      <TaskTable tasks={tasks} />
      <PaginationControls
        page={page}
        pageSize={pageSize}
        onPageChange={(p) => dispatch(setPage(p))}
        onPageSizeChange={(s) => dispatch(setPageSize(s))}
      />
    </Box>
  );
};

export default TasksPage;
