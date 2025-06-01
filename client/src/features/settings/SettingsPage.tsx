import React, { useEffect, useState } from 'react';
import {
  Box,
  Button,
  Heading,
  VStack,
  Menu,
  MenuButton,
  MenuList,
  Checkbox,
  MenuItem,
  Spinner,
} from '@chakra-ui/react';
// import { ChevronDownIcon } from '@chakra-ui/icons';
import { useDispatch } from 'react-redux';
import {
  setConnectionString,
  setDataForConnection,
  setSavedConnections,
} from './settingsSlice';
import { useNavigate } from 'react-router-dom';
import { useHttp } from '../../hooks/http.hook';

const SettingsPage: React.FC = () => {
  const dispatch = useDispatch();
  const navigate = useNavigate();
  const { request } = useHttp();

  const [selectedConnections, setSelectedConnections] = useState<string[]>([]);
  const [availableConnections, setAvailableConnections] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);

  const url = 'http://localhost:8888';

  const fetchConnections = async () => {
    setLoading(true);
    try {
      const rawData: Record<string, string> = await request(`${url}/api/get-connections`);
      const data: string[] = Object.values(rawData);
      setAvailableConnections(data);
      dispatch(setSavedConnections(data));
    } catch (e) {
      console.error('Ошибка при получении списка подключений:', e);
    } finally {
      setLoading(false);
    }
  };

  const handleToggle = (conn: string) => {
    setSelectedConnections((prev) =>
      prev.includes(conn)
        ? prev.filter((c) => c !== conn)
        : [...prev, conn]
    );
  };

  const handleChoose = async () => {
    if (selectedConnections.length === 0) return;

    const conn = selectedConnections[0];
    dispatch(setConnectionString(conn));

    try {
      const dbInfo = await request(`/api/get-data-base-info?conn=${encodeURIComponent(conn)}`);
      dispatch(setDataForConnection(dbInfo));
      navigate('/builder');
    } catch (e) {
      console.error('Ошибка при получении информации о БД:', e);
    }
  };

  useEffect(() => {
    fetchConnections();
  }, []);

  return (
    <Box maxW="lg" mx="auto" mt="20" p="8" borderWidth="1px" borderRadius="lg" boxShadow="lg">
      <VStack spacing={6} align="stretch">
        <Heading size="md" textAlign="center">Мастер генерации витрин</Heading>

        <Menu closeOnSelect={false}>
          <MenuButton as={Button} onClick={fetchConnections}>
            {selectedConnections.length > 0
              ? `${selectedConnections.length} выбрано`
              : 'Выбери подключения'}
          </MenuButton>
          <MenuList maxH="200px" overflowY="auto">
            {loading ? (
              <MenuItem><Spinner size="sm" /> Загрузка...</MenuItem>
            ) : (
              availableConnections.map((connStr, idx) => (
                <MenuItem key={idx} closeOnSelect={false}>
                  <Checkbox
                    isChecked={selectedConnections.includes(connStr)}
                    onChange={() => handleToggle(connStr)}
                  >
                    {connStr}
                  </Checkbox>
                </MenuItem>
              ))
            )}
          </MenuList>
        </Menu>

        <Button colorScheme="teal" onClick={handleChoose} isDisabled={selectedConnections.length === 0}>
          Сохранить и продолжить
        </Button>
      </VStack>
    </Box>
  );
};

export default SettingsPage;
