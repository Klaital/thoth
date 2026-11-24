import { useEffect } from "react";
import { useLocation } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import { trackUser, trackView } from "../telemetry";

/** Reports route changes and the signed-in user to frontend telemetry. */
export default function TelemetryTracker() {
  const { pathname } = useLocation();
  const { token } = useAuth();

  useEffect(() => trackView(pathname), [pathname]);
  useEffect(() => trackUser(token), [token]);

  return null;
}
