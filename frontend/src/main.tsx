import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import React from 'react'
import { BrowserRouter } from 'react-router-dom'
import { loadConfig } from './config.ts'

console.log("Page with profile: ", import.meta.env.VITE_PROFILE)
if (import.meta.env.VITE_PROFILE !== "prod") {
  void import('eruda').then(({ default: eruda }) => { eruda.init(); });
}



void loadConfig().then(() => {
  const rootElement = document.getElementById('root');
  if (!rootElement) throw new Error('Root element not found');
  createRoot(rootElement).render(
    <React.StrictMode>
      {/* Wrap the App component with BrowserRouter */}
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </React.StrictMode>,
  )
});
