import { useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../context/AuthContext";

// Landing page for the backend's Google OAuth callback, which redirects here
// with the token pair in the query string.
export default function AuthCallbackPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  // Handle the callback once. StrictMode runs effects twice in development,
  // and the first run's navigate() has already replaced the URL, so a second
  // run would find no tokens and bounce a signed-in user to /login.
  const handled = useRef(false);

  useEffect(() => {
    if (handled.current) return;
    handled.current = true;

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
