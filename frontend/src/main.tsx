import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import React from 'react'
import { BrowserRouter } from 'react-router-dom'
import { loadConfig } from './config.ts'

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore
console.log("Page with profile: ",import.meta.env.VITE_PROFILE)
if (import.meta.env.VITE_PROFILE !== "prod") {
  // eslint-disable-next-line @typescript-eslint/no-floating-promises
  import('eruda').then(({ default: eruda }) => { eruda.init(); });
}



// eslint-disable-next-line @typescript-eslint/no-floating-promises
loadConfig().then(() => {
  // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
  createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      {/* Wrap the App component with BrowserRouter */}
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </React.StrictMode>,
  )
});
