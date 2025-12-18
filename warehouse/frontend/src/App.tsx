import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { Navbar } from './components/Navbar';
import { Footer } from './components/Footer';
import { ProtectedRoute } from './components/ProtectedRoute';
import { HomePage } from './pages/HomePage';
import { LoginPage } from './pages/LoginPage';
import { DashboardPage } from './pages/DashboardPage';
import { RealmsListPage } from './pages/RealmsListPage';
import { RealmDetailPage } from './pages/RealmDetailPage';
import { AssetDetailPage } from './pages/AssetDetailPage';

function App() {
  return (
    <BrowserRouter basename={import.meta.env.VITE_BASE_PATH || '/'}>
      <AuthProvider>
        <div className="min-h-screen flex flex-col">
          <Navbar />
          <Routes>
            <Route path="/" element={<HomePage />} />
            <Route path="/login" element={<LoginPage />} />
            <Route
              path="/dashboard"
              element={
                <ProtectedRoute>
                  <DashboardPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/realms"
              element={
                <ProtectedRoute>
                  <RealmsListPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/realms/:realmId"
              element={
                <ProtectedRoute>
                  <RealmDetailPage />
                </ProtectedRoute>
              }
            />
            <Route
              path="/assets/:assetId"
              element={
                <ProtectedRoute>
                  <AssetDetailPage />
                </ProtectedRoute>
              }
            />
          </Routes>
          <Footer />
        </div>
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;