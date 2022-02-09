/// <reference types="react-scripts" />

// Env types.
declare namespace NodeJS {
  interface ProcessEnv {
    readonly REACT_APP_API_URL_PREFIX?: string
    readonly BUILD_TIMESTAMP: number
  }
}
