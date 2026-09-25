// src/api/axiosInstance.ts
import axios, { AxiosError } from 'axios';
import type { AxiosRequestConfig } from 'axios';
import { getConfig } from '../config';

const instance = axios.create({
  withCredentials: true,
});

// Use interceptor to dynamically set baseURL in case it's not set immediately,
// though in our setup loadConfig() runs before rendering, so it's guaranteed.
instance.interceptors.request.use((config) => {
  config.baseURL ??= getConfig().apiBaseUrl;

  if (config.url?.includes('/share/') || window.location.pathname.startsWith('/share')) {
    const pathParts = window.location.pathname.split('/');
    const id = pathParts[1] === 'share' ? pathParts[2] : null;
    if (id) {
      config.headers['X-Share-Id'] = id;
    }
  }

  return config;
});

// Single-flight refresh: concurrent 401s share one /refresh request instead of
// each spinning up its own (which would rotate the refresh token twice and trip
// the backend's reuse detection, revoking every session).
let refreshPromise: Promise<void> | null = null;

const refreshAccessToken = (): Promise<void> => {
  refreshPromise ??= instance
    .post('/refresh', null, { withCredentials: true })
    .then(() => undefined)
    .finally(() => {
      refreshPromise = null;
    });
  return refreshPromise;
};

interface CustomAxiosRequestConfig extends AxiosRequestConfig {
  _retry?: boolean;
}

instance.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as CustomAxiosRequestConfig | undefined;

    if (error.response?.status === 403) {
      const data = error.response.data as { error?: string } | undefined;
      if (data?.error === 'mfa_setup_required') {
        window.dispatchEvent(new Event('auth:mfa_setup_required'));
        if (!window.location.pathname.includes('/login')) {
           window.location.href = '/login?mfa_setup_required=true';
        }
        return Promise.reject(error);
      }
    }

    const isAuthEndpoint = Boolean(
      originalRequest?.url &&
      (originalRequest.url.includes('/refresh') ||
        originalRequest.url.includes('/login') ||
        originalRequest.url.includes('/mfa/verify') ||
        originalRequest.url.includes('/user/mfa/enable') ||
        originalRequest.url.includes('/logout'))
    );

    // If 401 and we haven't retried yet, and it's not an auth endpoint itself
    if (error.response?.status === 401 && originalRequest && !originalRequest._retry && !isAuthEndpoint) {
      if (window.location.pathname.startsWith('/share')) {
        window.dispatchEvent(new Event('share:unauthorized'));
        return Promise.reject(error);
      }

      // Mark before awaiting so a concurrent 401 for this same request cannot
      // kick off a second refresh/retry cycle.
      originalRequest._retry = true;

      try {
        await refreshAccessToken();
      } catch (refreshError: unknown) {
        window.dispatchEvent(new Event('auth:unauthorized'));
        return await Promise.reject(refreshError instanceof Error ? refreshError : new Error(String(refreshError)));
      }

      // Token refreshed — replay the original request exactly once.
      return await instance(originalRequest);
    }

    return Promise.reject(error);
  }
);

export default instance;