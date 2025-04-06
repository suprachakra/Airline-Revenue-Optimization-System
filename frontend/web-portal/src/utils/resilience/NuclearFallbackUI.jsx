import React from 'react';
import Alert from '@mui/material/Alert';

export function NuclearFallbackUI() {
  return (
    <div className="nuclear-fallback">
      <Alert severity="error">
        Emergency: Nuclear fallback mode activated. Some functionalities may be limited.
      </Alert>
      <p>Please contact support if this state persists.</p>
    </div>
  );
}
