import { useState, useEffect } from 'react';
import axios from 'axios';

/**
 * Custom hook to fetch real-time analytics data.
 * Sends performance and fallback events to both Google Analytics and Zenztech.
 */
export function useRealtimeAnalytics(endpoint, interval = 2000) {
  const [data, setData] = useState(null);
  const [status, setStatus] = useState('loading');
  const [error, setError] = useState(null);

  useEffect(() => {
    let isMounted = true;

    // Function to fetch data and log events if needed
    const fetchData = async () => {
      try {
        const startTime = Date.now();
        const response = await axios.get(endpoint);
        const latency = Date.now() - startTime;

        if (latency > 2000) { // if response is slow
          // Log slow response event
          if (window.gtag) {
            gtag('event', 'SlowResponse', {
              event_category: 'Performance',
              event_label: `Latency: ${latency}ms`
            });
          }
          if (window.zenztech) {
            zenztech('log', { event: 'SlowResponse', label: `Latency: ${latency}ms` });
          }
        }

        if (isMounted) {
          setData(response.data);
          setStatus('success');
        }
      } catch (err) {
        // Log error to analytics
        if (window.gtag) {
          gtag('event', 'AnalyticsError', {
            event_category: 'RealtimeAnalytics',
            event_label: err.message
          });
        }
        if (window.zenztech) {
          zenztech('log', { event: 'AnalyticsError', label: err.message });
        }
        if (isMounted) {
          setError(err);
          setStatus('error');
        }
      }
    };

    // Set an interval to repeatedly fetch analytics data
    const intervalId = setInterval(fetchData, interval);

    // Cleanup on component unmount
    return () => {
      isMounted = false;
      clearInterval(intervalId);
    };
  }, [endpoint, interval]);

  return { data, status, error };
}
