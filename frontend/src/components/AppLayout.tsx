import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const { logout } = useAuth();
  const navigate = useNavigate();

  function handleLogout() {
    logout();
    navigate("/login");
  }

  return (
    <>
      <nav className="app-nav">
        <Link to="/collections" className="brand">Thoth's Collection</Link>
        <button className="btn-ghost" onClick={handleLogout}>Sign out</button>
      </nav>
      {children}
    </>
  );
}
