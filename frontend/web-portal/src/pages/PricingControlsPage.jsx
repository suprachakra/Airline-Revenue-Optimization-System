import React, { useState } from 'react';
import { TextField, Button, Alert } from '@mui/material';

export default function PricingControlsPage() {
  const [overrideValue, setOverrideValue] = useState('');
  const [message, setMessage] = useState(null);

  const handleOverride = () => {
    if (Number(overrideValue) < 0 || Number(overrideValue) > 1000) {
      setMessage('Override value out of acceptable range.');
      return;
    }
    applyPricingOverride(overrideValue)
      .then(() => setMessage('Pricing override applied successfully.'))
      .catch(() => setMessage('Failed to apply override. Please try again.'));
  };

  return (
    <div className="pricing-controls">
      <h2>Manual Pricing Override</h2>
      {message && <Alert severity="info">{message}</Alert>}
      <TextField 
        label="Override Value" 
        value={overrideValue} 
        onChange={e => setOverrideValue(e.target.value)} 
      />
      <Button variant="contained" color="primary" onClick={handleOverride}>
        Apply Override
      </Button>
    </div>
  );
}
