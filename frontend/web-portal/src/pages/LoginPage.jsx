import React, { useState } from 'react';
import { TextField, Button, Alert } from '@mui/material';

export default function LoginPage() {
  const [error, setError] = useState(null);
  
  const handleLogin = async (e) => {
    e.preventDefault();
    try {
      // Authenticate user via API
      await authenticateUser(e.target.username.value, e.target.password.value);
    } catch (err) {
      setError(err);
    }
  };

  return (
    <div className="login-container">
      <h1>Login</h1>
      {error && <Alert severity="error">{error.message}</Alert>}
      <form onSubmit={handleLogin}>
        <TextField label="Username" name="username" required />
        <TextField label="Password" name="password" type="password" required />
        <Button type="submit" variant="contained" color="primary">Login</Button>
      </form>
    </div>
  );
}
