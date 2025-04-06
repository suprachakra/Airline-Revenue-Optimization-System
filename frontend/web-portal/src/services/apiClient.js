import axios from 'axios';
import { CircuitBreaker } from '@resilience/patterns';
import { getAuthToken, refreshToken } from './authService';

// Initialize the Circuit Breaker with fallback function that logs the event to both GA and Zenztech.
const circuitBreaker = new CircuitBreaker({
  threshold: 5,             // 5 consecutive failures trip the circuit
  timeout: 30000,           // 30s reset window
  fallback: () => {
    // Log the fallback event to Google Analytics and Zenztech
    if (window.gtag) {
      gtag('event', 'FallbackTriggered', {
        event_category: 'CircuitBreaker',
        event_label: 'nuclear_fallback'
      });
    }
    if (window.zenztech) {
      zenztech('log', { event: 'FallbackTriggered', label: 'nuclear_fallback' });
    }
    return {
      data: loadCriticalCache('emergency_data'),
      headers: { 'X-Fallback': 'nuclear' }
    };
  }
});

export const apiClient = axios.create({
  baseURL: process.env.REACT_APP_API_BASE,
  timeout: 10000, // 10s timeout for API requests
  headers: {
    'Content-Security-Policy': "default-src 'self' api.iaros.ai"
  }
});

// Intelligent Retry and Error Interceptor with Analytics Logging
apiClient.interceptors.response.use(
  response => response,
  async error => {
    const originalRequest = error.config;
    
    // Handle token expiration
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;
      const newToken = await refreshToken();
      apiClient.defaults.headers.common['Authorization'] = `Bearer ${newToken}`;
      return apiClient(originalRequest);
    }

    // Handle network errors with Circuit Breaker integration
    if (error.code === 'ECONNABORTED' || !error.response) {
      // Log the network error event to analytics
      if (window.gtag) {
        gtag('event', 'NetworkError', {
          event_category: 'API',
          event_label: 'TimeoutOrNoResponse'
        });
      }
      if (window.zenztech) {
        zenztech('log', { event: 'NetworkError', label: 'TimeoutOrNoResponse' });
      }
      return circuitBreaker.execute(() => apiClient(originalRequest));
    }

    // Default error handling
    return Promise.reject(error);
  }
);

// Request Interceptor for Offline Mode - serves cached data if offline.
apiClient.interceptors.request.use(config => {
  if (!navigator.onLine) {
    return Promise.resolve({
      data: loadPersistentCache(config.url),
      status: 200,
      headers: { 'X-Cache': 'persistent' },
      config
    });
  }
  return config;
});
