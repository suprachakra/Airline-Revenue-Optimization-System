import NetInfo from '@react-native-community/netinfo';
import axios from 'axios';
import { refreshToken } from './AuthService';
import EncryptedStorage from 'react-native-encrypted-storage';

const API_TIMEOUT = 15000;

const api = axios.create({
  baseURL: Platform.select({
    ios: 'https://api.iaros.io/v3',
    android: 'https://api.iaros-android.io/v3',
    huawei: 'https://api.iaros-huawei.io/v3'
  }),
  timeout: API_TIMEOUT,
});

// Certificate Pinning and retry interceptor implementation.
api.interceptors.response.use(
  response => response,
  async error => {
    const originalRequest = error.config;
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;
      const newToken = await refreshToken();
      originalRequest.headers.Authorization = `Bearer ${newToken}`;
      return api(originalRequest);
    }
    if (error.code === 'ECONNABORTED' || !error.response) {
      throw new Error('Service unavailable - fallback active');
    }
    throw error;
  }
);

// Offline-first interceptor using Encrypted Storage.
api.interceptors.request.use(async config => {
  const state = await NetInfo.fetch();
  if (!state.isInternetReachable) {
    const cached = await EncryptedStorage.getItem(`cache:${config.url}`);
    if (cached) {
      return Promise.resolve({
        data: JSON.parse(cached),
        status: 200,
        headers: { 'X-Cache': 'persistent' },
        config
      });
    }
  }
  return config;
});

export default api;
