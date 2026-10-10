import { Routes, Route, Navigate } from "react-router-dom";
import AuthCallbackPage from "./pages/AuthCallbackPage";
import LoginPage from "./pages/LoginPage";
import RegisterPage from "./pages/RegisterPage";
import CollectionsPage from "./pages/CollectionsPage";
import CollectionDetailPage from "./pages/CollectionDetailPage";
import NewCollectionPage from "./pages/NewCollectionPage";
import AddItemPage from "./pages/AddItemPage";
import { useAuth } from "./context/AuthContext";

function RequireAuth({ children }: { children: React.ReactNode }) {
  const { token } = useAuth();
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function App() {
  return (
    <Routes>
      <Route path="/auth/callback" element={<AuthCallbackPage />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route
        path="/collections"
        element={
          <RequireAuth>
            <CollectionsPage />
          </RequireAuth>
        }
      />
      <Route
        path="/collections/new"
        element={
          <RequireAuth>
            <NewCollectionPage />
          </RequireAuth>
        }
      />
      <Route
        path="/collections/:id"
        element={
          <RequireAuth>
            <CollectionDetailPage />
          </RequireAuth>
        }
      />
      <Route
        path="/collections/:id/add"
        element={
          <RequireAuth>
            <AddItemPage />
          </RequireAuth>
        }
      />
      <Route path="/" element={<Navigate to="/collections" replace />} />
    </Routes>
  );
}
