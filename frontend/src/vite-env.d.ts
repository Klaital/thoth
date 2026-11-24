/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** Grafana Faro collector URL; frontend telemetry is off when unset. */
  readonly VITE_FARO_URL?: string;
  readonly VITE_APP_VERSION?: string;
  readonly VITE_APP_ENV?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
