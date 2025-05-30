import React, {useState} from 'react';
import {Box, Button, Heading, Select, VStack} from '@chakra-ui/react';
import {useDispatch} from 'react-redux';
import {setConnectionString, setDataForConnection, setSavedConnections} from './settingsSlice';
import {useNavigate} from 'react-router-dom';
import {useHttp} from '../../hooks/http.hook';

const SettingsPage : React.FC = () => {
    const dispatch = useDispatch();
    const navigate = useNavigate();
    const {request} = useHttp();

    const [conn,
        setConn] = useState < string > ('');
    const [availableConnections,
        setAvailableConnections] = useState < string[] > ([]);

    const handleOpenSelect = async() => {
        if (availableConnections.length === 0) {
            try {
                const data : string[] = await request('/api/get-connections');
                setAvailableConnections(data);
                dispatch(setSavedConnections(data));
            } catch (e) {
                console.error('Ошибка при получении списка подключений:', e);
            }
        }
    };

    const handleChoose = async() => {
        if (!conn.trim()) 
            return;
        
        dispatch(setConnectionString(conn));

        try {
            const dbInfo = await request(`/api/get-data-base-info?conn=${encodeURIComponent(conn)}`);
            dispatch(setDataForConnection(dbInfo));
            navigate('/builder');
        } catch (e) {
            console.error('Ошибка при получении информации о БД:', e);
        }
    };

    return (
        <Box
            maxW="lg"
            mx="auto"
            mt="20"
            p="8"
            borderWidth="1px"
            borderRadius="lg"
            boxShadow="lg">
            <VStack spacing={6} align="stretch">
                <Heading size="md" textAlign="center">Настройки подключения</Heading>
                <Select
                    placeholder="Выбери из сохранённых"
                    onClick={handleOpenSelect}
                    onChange={(e) => setConn(e.target.value)}
                    value={conn}>
                    {availableConnections.map((connStr, idx) => (
                        <option key={idx} value={connStr}>
                            {connStr}
                        </option>
                    ))}
                </Select>
                <Button colorScheme="teal" onClick={handleChoose}>
                    Сохранить и продолжить
                </Button>
            </VStack>
        </Box>
    );
};

export default SettingsPage;
