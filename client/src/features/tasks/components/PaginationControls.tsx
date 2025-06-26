import React from 'react';
import { HStack, Button, Select } from '@chakra-ui/react';
import { useSelector } from 'react-redux';
import type { RootState } from '../../../app/store';

interface Props {
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
}

const PaginationControls: React.FC<Props> = ({
  page,
  pageSize,
  onPageChange,
  onPageSizeChange,
}) => {
    const { tasks} = useSelector((state: RootState) => state.tasks);
        console.log(pageSize)

    console.log(tasks.length)
  return (
  <HStack mt={4} spacing={4} justify="center">
    <Button onClick={() => onPageChange(page - 1)} isDisabled={page <= 1}>
      Предыдущая
    </Button>
    <Select
      value={pageSize}
      onChange={(e) => onPageSizeChange(Number(e.target.value))}
      width="auto"
    >
      {[5, 10, 20, 50].map((size) => (
        <option key={size} value={size}>
          {size} на странице
        </option>
      ))}
    </Select>
    <Button onClick={() => onPageChange(page + 1)} isDisabled={pageSize > tasks.length}>Следующая</Button>
  </HStack>
);


}
export default PaginationControls;
