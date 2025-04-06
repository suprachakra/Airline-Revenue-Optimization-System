import React from 'react';
import { useCookieConsent } from '../hooks/useCookieConsent';
import ConsentBanner from '../components/ConsentBanner';

export const GDPRGuard = ({ children }) => {
  const [consent, setConsent] = useCookieConsent();
  return consent.level === 'full' ? children : <ConsentBanner onAccept={setConsent} />;
};
