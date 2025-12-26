import React, {useEffect, useMemo, useState} from 'react';
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
    Tooltip
} from '@chakra-ui/react';
// import { ChevronDownIcon } from '@chakra-ui/icons';
import {useDispatch} from 'react-redux';
import {
    setConnectionString,
    setDataForConnection,
    setSavedConnections,
    setSelectedConnections,
} from './settingsSlice';
import {useNavigate} from 'react-router-dom';
import {useHttp} from '../../hooks/http.hook';

const SettingsPage : React.FC = () => {
    const dispatch = useDispatch();
    const navigate = useNavigate();
    const {request} = useHttp();

    const [selectedConnections,
        setSelectedConnectionsState] = useState < string[] > ([]);
    const [availableConnections,
        setAvailableConnections] = useState<Record<string, string>>({});
    const [loading,
        setLoading] = useState(false);

    const url = '/api';
    const fetchConnections = async() => {
        setLoading(true);
        try {
            const rawData = await request(`${url}/get-connections`);
            setAvailableConnections(rawData);
            dispatch(setSavedConnections(rawData));
        } catch (e) {
            console.error('Ошибка при получении списка подключений:', e);
        } finally {
            setLoading(false);
        }
    };

    const handleToggle = (conn : string) => {
        setSelectedConnectionsState((prev) => prev.includes(conn)
            ? prev.filter((c) => c !== conn)
            : [
                ...prev,
                conn
            ]);
    };

    const chosenConnectionStrings = useMemo(
        () =>
            selectedConnections
                .map((key) => ({
                    key,
                    value: availableConnections[key],
                }))
                .filter((item): item is { key: string; value: string } => Boolean(item.value)),
        [availableConnections, selectedConnections],
    );

    const handleChoose = async() => {
        if (chosenConnectionStrings.length === 0)
            return;

        const firstConnection = chosenConnectionStrings[0]?.value ?? '';
        dispatch(setConnectionString(firstConnection));
        dispatch(setSelectedConnections(chosenConnectionStrings.map(({key}) => key)));

        try {
        const body = {
            connection_strings: chosenConnectionStrings.map(({key, value}) => ({
                connection_string: {
                    [key]: value,
                },
            })),
            page: 1,
            page_size: 20,
        };
        const dbInfo = await request(`${url}/get-db`, "POST", body);
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
        <Box
            maxW="lg"
            mx="auto"
            mt="20"
            p="8"
            borderWidth="1px"
            borderRadius="lg"
            boxShadow="lg">
            <VStack spacing={6} align="stretch">
                <Heading size="md" textAlign="center">Мастер генерации витрин</Heading>

                <Menu closeOnSelect={false}>
                    <MenuButton as={Button} onClick={fetchConnections}>
                        {selectedConnections.length > 0
                            ? `${selectedConnections.length} выбрано`
                            : 'Выбери подключения'}
                    </MenuButton>
                    <MenuList maxH="200px" overflowY="auto">
                        {loading
                            ? (
                                <MenuItem><Spinner size="sm"/>
                                    Загрузка...</MenuItem>
                            )
                            : (Object.entries(availableConnections).map(([key, connStr]) => (
                                <MenuItem key={key} closeOnSelect={false}>
                                    <Tooltip
                                        label={connStr}
                                        hasArrow
                                        placement="top"
                                        closeOnClick={true}
                                        closeDelay={100}>
                                        <Box w="100%">
                                            <Checkbox
                                                isChecked={selectedConnections.includes(key)}
                                                onChange={() => handleToggle(key)}>
                                                {key}
                                            </Checkbox>
                                        </Box>
                                    </Tooltip>
                                </MenuItem>
                            )))}
                    </MenuList>
                </Menu>

                <Button
                    colorScheme="teal"
                    onClick={handleChoose}
                    isDisabled={selectedConnections.length === 0}>
                    Сохранить и продолжить
                </Button>
            </VStack>
        </Box>
    );
};

export default SettingsPage;
