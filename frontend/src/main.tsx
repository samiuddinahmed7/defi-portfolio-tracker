import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './styles/index.css'
import { Dashboard } from './pages/Dashboard'

const container = document.getElementById('root')!
createRoot(container).render(
  <StrictMode>
    <Dashboard />
  </StrictMode>,
)
