import { useEffect, useMemo, useRef } from 'react';
import { useToast } from '@chakra-ui/react';

type NotificationMessage = {
  type: string;
  title: string;
  message: string;
  taskId?: string;
  status?: string;
  createdAt?: string;
};

const buildWebSocketUrl = () => {
  const base = import.meta.env.VITE_API_BASE_URL;

  if (base) {
    const url = new URL(base);
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    const trimmedPath = url.pathname.endsWith('/') ? url.pathname.slice(0, -1) : url.pathname;
    url.pathname = `${trimmedPath}/ws/notifications`;
    return url.toString();
  }

  const { protocol, host } = window.location;
  const wsProtocol = protocol === 'https:' ? 'wss:' : 'ws:';
  return `${wsProtocol}//${host}/ws/notifications`;
};

const NotificationCenter = () => {
  const toast = useToast();
  const socketRef = useRef<WebSocket | null>(null);

  const wsUrl = useMemo(() => buildWebSocketUrl(), []);

  useEffect(() => {
    const socket = new WebSocket(wsUrl);
    socketRef.current = socket;

    socket.onmessage = (event) => {
      try {
        const notification: NotificationMessage = JSON.parse(event.data);
        toast({
          title: notification.title || 'Уведомление',
          description: notification.message,
          status: notification.status?.toLowerCase() === 'completed' ? 'success' : 'info',
          duration: 6000,
          isClosable: true,
          position: 'top-right',
        });
      } catch (error) {
        console.error('Failed to parse notification', error);
      }
    };

    socket.onerror = (error) => {
      console.error('WebSocket error', error);
    };

    return () => {
      socket.close();
      socketRef.current = null;
    };
  }, [toast, wsUrl]);

  return null;
};

export default NotificationCenter;
