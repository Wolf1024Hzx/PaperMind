import { useState, useCallback } from 'react';
import { ApiErrorClass } from '../api/client';
import toast from 'react-hot-toast';

interface UseApiOptions {
  onSuccess?: (data: unknown) => void;
  onError?: (error: ApiErrorClass) => void;
}

export function useApi<T>(options?: UseApiOptions) {
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<ApiErrorClass | null>(null);
  const [data, setData] = useState<T | null>(null);

  const execute = useCallback(
    async (apiCall: () => Promise<T>) => {
      setIsLoading(true);
      setError(null);

      try {
        const result = await apiCall();
        setData(result);
        options?.onSuccess?.(result);
        return result;
      } catch (err) {
        const apiError =
          err instanceof ApiErrorClass
            ? err
            : new ApiErrorClass(500, { message: '未知错误' });
        setError(apiError);
        options?.onError?.(apiError);
        toast.error(apiError.data.message);
        throw apiError;
      } finally {
        setIsLoading(false);
      }
    },
    [options]
  );

  return {
    isLoading,
    error,
    data,
    execute,
    reset: () => {
      setIsLoading(false);
      setError(null);
      setData(null);
    },
  };
}