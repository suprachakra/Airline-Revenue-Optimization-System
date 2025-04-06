import React from 'react';
import { Alert } from 'react-native';
import FingerprintScanner from 'react-native-fingerprint-scanner';

export default function BiometricFallback({ onSuccess, onFailure }) {
  const handleFallback = async () => {
    try {
      const result = await FingerprintScanner.authenticate({
        description: 'Authenticate to access IAROS',
        fallbackEnabled: true
      });
      onSuccess(result);
    } catch (error) {
      Alert.alert('Authentication Failed', 'Please enter your PIN as a backup.');
      onFailure(error);
    }
  };

  return (
    <Alert onPress={handleFallback} title="Biometric authentication failed, tap to fallback to PIN." />
  );
}
