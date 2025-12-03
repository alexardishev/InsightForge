import { useState, useCallback } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '';

const buildUrl = (url: string) => {
  if (/^https?:\/\//i.test(url)) {
    return url;
  }

  if (!API_BASE_URL) {
    return url;
  }

  const needsSlash = !API_BASE_URL.endsWith('/') && !url.startsWith('/');
  return `${API_BASE_URL}${needsSlash ? '/' : ''}${url}`;
};

export const useHttp = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const request = useCallback(
    async <T = any>(
      url: string,
      method = 'GET',
      body: any = null,
      headers = { 'Content-Type': 'application/json' }
    ): Promise<T> => {
      setLoading(true);
      setError(null);
      try {
        const response = await fetch(buildUrl(url), {
          method,
          body: body ? JSON.stringify(body) : null,
          headers,
        });

        if (!response.ok) {
          throw new Error(`Ошибка запроса: ${response.status}`);
        }

        const data = await response.json();
        return data as T;
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
