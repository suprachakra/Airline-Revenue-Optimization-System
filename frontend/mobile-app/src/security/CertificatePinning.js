import { fetch } from 'react-native-ssl-pinning';

export const secureFetch = async (url, options = {}) => {
  return fetch(url, {
    ...options,
    timeoutInterval: 15000,
    sslPinning: {
      certs: ['sha256/ExampleYourPublicKeyHash']
    }
  });
};
