// Service Worker for IAROS Web Portal - Offline-first caching and background sync.
import { precacheAndRoute } from 'workbox-precaching';
import { NetworkFirst, StaleWhileRevalidate } from 'workbox-strategies';
import { BackgroundSyncPlugin } from 'workbox-background-sync';

// Precache critical assets.
precacheAndRoute(self.__WB_MANIFEST);

// Background Sync Plugin for API requests.
const bgSync = new BackgroundSyncPlugin('apiQueue', {
  maxRetentionTime: 48 * 60 // 48 hours
});

// API Request Strategy: Network First with Background Sync.
registerRoute(
  ({url}) => url.pathname.startsWith('/api/'),
  new NetworkFirst({
    cacheName: 'api-cache',
    plugins: [bgSync]
  }),
  'GET'
);

// Static Assets Strategy: Stale While Revalidate.
registerRoute(
  ({request}) => request.destination === 'script' || request.destination === 'style',
  new StaleWhileRevalidate({
    cacheName: 'static-assets'
  })
);
