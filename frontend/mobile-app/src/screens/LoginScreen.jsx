import React, { useState } from 'react';
import { View, TextInput, Button, Text, StyleSheet } from 'react-native';

export default function LoginScreen() {
  const [error, setError] = useState(null);

  const handleLogin = async () => {
    // Implement mobile-specific authentication.
    try {
      await authenticateUser();
    } catch (err) {
      setError(err);
    }
  };

  return (
    <View style={styles.container}>
      {error && <Text style={styles.errorText}>{error.message}</Text>}
      <TextInput style={styles.input} placeholder="Username" />
      <TextInput style={styles.input} placeholder="Password" secureTextEntry />
      <Button title="Login" onPress={handleLogin} />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { padding: 20 },
  input: { borderWidth: 1, marginBottom: 10, padding: 8 },
  errorText: { color: 'red', marginBottom: 10 }
});
