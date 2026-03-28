/// <reference lib="webworker" />
import { clientsClaim } from 'workbox-core'
import { precacheAndRoute, cleanupOutdatedCaches } from 'workbox-precaching'
import { registerRoute } from 'workbox-routing'
import { StaleWhileRevalidate } from 'workbox-strategies'

declare const self: ServiceWorkerGlobalScope

// Take control of existing clients immediately
clientsClaim()

// Precache manifest injected by VitePWA
precacheAndRoute(self.__WB_MANIFEST)

// Clean up old cache versions
cleanupOutdatedCaches()

// Cache same-origin navigation requests (SPA fallback)
registerRoute(
  ({ request }) => request.mode === 'navigate',
  new StaleWhileRevalidate({
    cacheName: 'pages',
  }),
)

// Cache CSS/JS/API assets
registerRoute(
  ({ request }) =>
    ['style', 'script', 'worker'].includes(request.destination),
  new StaleWhileRevalidate({
    cacheName: 'assets',
  }),
)
