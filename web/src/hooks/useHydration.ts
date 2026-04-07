import { useAuthStore } from '../stores/authStore';
import { useEffect, useState } from 'react';

export function useHydration() {
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    // Zustand persist hydration happens synchronously on client
    setHydrated(useAuthStore.persist.hasHydrated());
  }, []);

  return hydrated;
}