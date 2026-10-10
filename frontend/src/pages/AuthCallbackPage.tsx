import { useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

export default function AuthCallbackPage() {
  const { login } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const accessToken = params.get("access_token");
    const refreshToken = params.get("refresh_token");

    if (accessToken && refreshToken) {
      login(accessToken, refreshToken);
      navigate("/collections", { replace: true });
    } else {
      navigate("/login", { replace: true });
    }
  }, [login, navigate]);

  return <p>Signing you in…</p>;
}
