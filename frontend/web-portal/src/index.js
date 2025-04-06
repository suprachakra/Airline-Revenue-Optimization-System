import React from 'react';
import ReactDOM from 'react-dom';
import App from './App';
import './styles/main.css';
import * as serviceWorkerManager from './utils/resilience/ServiceWorkerManager';

ReactDOM.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
  document.getElementById('root')
);

// Register service worker for offline support.
serviceWorkerManager;
