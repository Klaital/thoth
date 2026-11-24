import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
import "./index.css";
import { AuthProvider } from "./context/AuthContext";
import TelemetryTracker from "./components/TelemetryTracker";
import { initTelemetry } from "./telemetry";

initTelemetry();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <BrowserRouter>
      <AuthProvider>
        <TelemetryTracker />
        <App />
      </AuthProvider>
    </BrowserRouter>
  </StrictMode>
);
