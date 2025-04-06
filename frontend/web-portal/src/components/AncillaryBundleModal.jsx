import React from 'react';
import Modal from '@mui/material/Modal';
import Button from '@mui/material/Button';

export default function AncillaryBundleModal({ open, onClose, onSave, bundleOptions }) {
  return (
    <Modal open={open} onClose={onClose}>
      <div className="modal-container">
        <h2>Configure Ancillary Bundle</h2>
        <ul>
          {bundleOptions.map(option => (
            <li key={option.id}>{option.name}</li>
          ))}
        </ul>
        <div className="modal-actions">
          <Button variant="contained" color="primary" onClick={onSave}>
            Save Bundle
          </Button>
          <Button variant="outlined" onClick={onClose}>
            Cancel
          </Button>
        </div>
        {/* Fallback: Display a message if service calls fail */}
      </div>
    </Modal>
  );
}
