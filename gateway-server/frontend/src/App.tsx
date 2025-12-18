import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import LoginPage from './pages/LoginPage';
import VirtualServicesPage from './pages/VirtualServicesPage';
import CreateServicePage from './pages/CreateServicePage';
import VSDetailPage from './pages/VSDetailPage';

function App() {
  return (
    <BrowserRouter basename={import.meta.env.VITE_BASE_PATH || '/'}>
      <AuthProvider>
        <Routes>
          <Route path="/" element={<LoginPage />} />
          <Route path="/services" element={<VirtualServicesPage />} />
          <Route path="/services/new" element={<CreateServicePage />} />
          <Route path="/services/:vsId" element={<VSDetailPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;