import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { Toast } from 'radix-ui'
import { ThemeProvider } from '@/components/theme-provider'
import { TooltipProvider } from '@/components/ui/tooltip'
import { ToasterViewport } from '@/components/ui/toaster'
import './index.css'
import App from './App.tsx'

// Ошибка загрузки чанков (например после деплоя) — подсказка в консоли
window.addEventListener('unhandledrejection', (event) => {
  const msg = String(event.reason?.message ?? event.reason ?? '')
  if (msg.includes('Failed to fetch dynamically imported module') || msg.includes('Loading chunk')) {
    console.warn('[kb] Chunk load error — try hard refresh (Ctrl+Shift+R):', msg)
  }
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Toast.Provider swipeDirection="up" duration={5000} label="Уведомления">
      <ThemeProvider defaultTheme="system" enableSystem disableTransitionOnChange attribute="class">
        <TooltipProvider>
          <BrowserRouter>
            <App />
          </BrowserRouter>
          <ToasterViewport />
        </TooltipProvider>
      </ThemeProvider>
    </Toast.Provider>
  </StrictMode>,
)
