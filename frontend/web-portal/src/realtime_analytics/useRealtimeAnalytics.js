import { useState, useEffect } from 'react';

export function useRealtimeAnalytics() {
  const [dataFreshness, setDataFreshness] = useState(true);
  useEffect(() => {
    const interval = setInterval(() => {
      const lastUpdate = parseInt(localStorage.getItem('lastUpdate') || 0, 10);
      const isFresh = Date.now() - lastUpdate < 2000;
      setDataFreshness(isFresh);
    }, 1000);
    return () => clearInterval(interval);
  }, []);
  return { dataFreshness };
}
