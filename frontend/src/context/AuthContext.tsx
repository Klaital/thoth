import { createContext, useCallback, useContext, useState } from "react";
import { clearTokens, getAccessToken, setTokens } from "../api/auth";

interface AuthContextValue {
  token: string | null;
  login: (access: string, refresh: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setToken] = useState<string | null>(getAccessToken);

  const login = useCallback((access: string, refresh: string) => {
    setTokens(access, refresh);
    setToken(access);
  }, []);

  const logout = useCallback(() => {
    clearTokens();
    setToken(null);
  }, []);

  return (
    <AuthContext.Provider value={{ token, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside AuthProvider");
  return ctx;
}
