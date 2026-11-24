// Frontend telemetry via Grafana Faro: page views/sessions, Web Vitals, JS
// errors, and trace propagation on fetch so browser activity links to backend
// traces. Entirely disabled (and the SDK never downloaded) unless
// VITE_FARO_URL is set at build time.

type FaroModule = typeof import("@grafana/faro-web-sdk");

const faroUrl = import.meta.env.VITE_FARO_URL;
let faro: Promise<FaroModule["faro"] | null> = Promise.resolve(null);

export function initTelemetry(): void {
  if (!faroUrl) return;
  faro = Promise.all([
    import("@grafana/faro-web-sdk"),
    import("@grafana/faro-web-tracing"),
  ])
    .then(([sdk, tracing]) => {
      sdk.initializeFaro({
        url: faroUrl,
        app: {
          name: "thoth-web",
          version: import.meta.env.VITE_APP_VERSION,
          environment: import.meta.env.VITE_APP_ENV ?? "prod",
        },
        // Routes are reported by trackView below with IDs stripped.
        instrumentations: [
          ...sdk.getWebInstrumentations(),
          new tracing.TracingInstrumentation(),
        ],
      });
      return sdk.faro;
    })
    .catch((err) => {
      console.warn("telemetry init failed", err);
      return null;
    });
}

/** Collapses numeric path segments so views group by route, not by record. */
export function routeName(pathname: string): string {
  return pathname.replace(/\/\d+(?=\/|$)/g, "/:id") || "/";
}

export function trackView(pathname: string): void {
  void faro.then((f) => f?.api.setView({ name: routeName(pathname) }));
}

/** Reads the user ID claim from a JWT without verifying it (display only). */
export function userIdFromToken(token: string | null): string | null {
  if (!token) return null;
  try {
    const payload = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const uid = JSON.parse(atob(payload)).uid;
    return uid == null ? null : String(uid);
  } catch {
    return null;
  }
}

/** Tags the session with the user ID only — no email or other PII. */
export function trackUser(token: string | null): void {
  const id = userIdFromToken(token);
  void faro.then((f) => {
    if (!f) return;
    if (id) f.api.setUser({ id });
    else f.api.resetUser();
  });
}
