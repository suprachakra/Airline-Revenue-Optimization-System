import React, { useEffect } from 'react';
import axe from 'axe-core';

export default function AccessibilityWatcher() {
  useEffect(() => {
    axe.run(document, (err, results) => {
      if (err) throw err;
      if (results.violations.length > 0) {
        console.warn('Accessibility issues detected:', results.violations);
      }
    });
  }, []);
  return null;
}
