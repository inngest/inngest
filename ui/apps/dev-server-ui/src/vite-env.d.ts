/// <reference types="vite/client" />

interface ImportMetaEnv {
  // set by vite.config.ts from INNGEST_UI_HOSTED so browser code and
  // prerender use the same hosted mode.
  readonly VITE_HOSTED?: string;
  readonly VITE_PUBLIC_API_BASE_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
