import React from 'react';
import { HStack, Button, Select } from '@chakra-ui/react';

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
}) => (
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
    <Button onClick={() => onPageChange(page + 1)}>Следующая</Button>
  </HStack>
);

export default PaginationControls;
