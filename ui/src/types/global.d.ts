export {};

declare global {
  interface Window {
    /**
     * Injected by the native shell (iOS/Android) at startup.
     * Absent in browser-based development mode.
     */
    __OPENMENSES_ENGINE__?: {
      /** TCP port the Go engine's HTTP server is listening on. */
      port: number;
      /** Bearer token required for Connect-RPC requests. */
      authToken: string;
      /** Whether HealthKit sync is currently enabled (injected by native shell). */
      healthKitSyncEnabled: boolean;
    };
  }
}
