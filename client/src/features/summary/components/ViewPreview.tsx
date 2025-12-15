import React from 'react';
import { Box, Heading, Text } from '@chakra-ui/react';
import JsonViewer from '../../../components/JsonViewer';

interface Props {
  view: any; // TODO: define proper type
}

const ViewPreview: React.FC<Props> = ({ view }) => (
  <Box>
    <Heading size="md" mb={2}>Структура view</Heading>
    <Text color="text.muted" mb={4}>
      Подробный JSON с таблицами, джоинами и трансформациями. Разворачивай узлы, чтобы проверить детали.
    </Text>
    <JsonViewer data={view} />
  </Box>
);

export default ViewPreview;
