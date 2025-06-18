import React from 'react';
import { Button } from '@chakra-ui/react';
import { useNavigate } from 'react-router-dom';
import { useHttp } from '../../../hooks/http.hook';

interface Props {
  view: any; // TODO: define proper type
}

const SummaryActions: React.FC<Props> = ({ view }) => {
  const navigate = useNavigate();
  const { request } = useHttp();

  const handleSend = async () => {
    try {
      await request('http://localhost:8888/api/upload-schem', 'POST', view);
      navigate('/settings');
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <>
      <Button colorScheme="teal" onClick={handleSend} mr={4}>
        Отправить
      </Button>
      <Button onClick={() => navigate('/transforms')}>Назад</Button>
    </>
  );
};

export default SummaryActions;
