import React from 'react';
import { Textarea } from '@chakra-ui/react';

interface Props {
  view: any; // TODO: define proper type
}

const ViewPreview: React.FC<Props> = ({ view }) => (
  <Textarea value={JSON.stringify(view, null, 2)} readOnly h="400px" mb={4} />
);

export default ViewPreview;
