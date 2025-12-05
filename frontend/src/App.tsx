import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from './lib/queryClient'
import DotGridBackground from './components/layout/DotGridBackground'
import HeaderDock from './components/layout/HeaderDock'
import DashboardPage from './pages/DashboardPage'
import DevicesPage from './pages/DevicesPage'
import TunnelsPage from './pages/TunnelsPage'
import ServicesPage from './pages/ServicesPage'
import TimelinePage from './pages/TimelinePage'
import SettingsPage from './pages/SettingsPage'

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
        <DotGridBackground>
          <HeaderDock />
          <main className="app-main-content">
            <Routes>
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
              <Route path="/dashboard" element={<DashboardPage />} />
              <Route path="/devices" element={<DevicesPage />} />
              <Route path="/tunnels" element={<TunnelsPage />} />
              <Route path="/services" element={<ServicesPage />} />
              <Route path="/timeline" element={<TimelinePage />} />
              <Route path="/settings" element={<SettingsPage />} />
            </Routes>
          </main>
        </DotGridBackground>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
