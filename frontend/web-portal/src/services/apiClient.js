import axios from 'axios';
import { CircuitBreaker } from '@resilience/patterns';
import { getAuthToken, refreshToken } from './authService';
import { loadCriticalCache } from '../utils/cacheUtils';

const circuitBreaker = new CircuitBreaker({
  threshold: 5, // 5 consecutive failures trigger circuit open state
  timeout: 30000, // 30-second reset window
  fallback: () => ({
    data: loadCriticalCache('emergency_data'),
    headers: { 'X-Fallback': 'nuclear' }
  })
});

const api = axios.create({
  baseURL: process.env.REACT_APP_API_BASE,
  timeout: 10000, // 10-second timeout
  headers: {
    'Content-Security-Policy': "default-src 'self' api.iaros.ai"
  }
});

// Intelligent Retry Interceptor
api.interceptors.response.use(null, async (error) => {
  const config = error.config;
  config._retryCount = config._retryCount || 0;

  // Token Rotation for 401 errors
  if (error.response?.status === 401 && !config._retry) {
    config._retry = true;
    const newToken = await refreshToken();
    api.defaults.headers.common['Authorization'] = `Bearer ${newToken}`;
    return api(config);
  }

  // Circuit Breaker integration for network errors or timeouts
  if (error.code === 'ECONNABORTED' || !error.response) {
    return circuitBreaker.execute(() => api(config));
  }

  return Promise.reject(error);
});

// Critical Request Monitor: Serve cached data when offline
api.interceptors.request.use(config => {
  if (!navigator.onLine) {
    return Promise.resolve({
      data: loadCriticalCache(config.url),
      status: 200,
      headers: { 'X-Cache': 'persistent' },
      config
    });
  }
  return config;
});

export const secureRequest = async (config) => {
  return circuitBreaker.execute(() => api(config));
};

export default api;
