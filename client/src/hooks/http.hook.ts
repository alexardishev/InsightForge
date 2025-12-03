import { useState, useCallback } from 'react';

export const useHttp = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const request = useCallback(
    async (url: string, method = 'GET', body: any = null, headers = { 'Content-Type': 'application/json' }) => {
      setLoading(true);
      setError(null);
      try {
        const response = await fetch(url, {
          method,
          body: body ? JSON.stringify(body) : null,
          headers,
        });

        if (!response.ok) {
          throw new Error(`Ошибка запроса: ${response.status}`);
        }

        const data = await response.json();
        return data;
      } catch (e: any) {
        setError(e.message);
        throw e;
      } finally {
        setLoading(false);
      }
    },
    []
  );

  return { request, loading, error };
};
