/**
 * Utilities for communicating with the iOS native HealthKit integration.
 * These functions post messages to the WKScriptMessageHandler registered
 * under the "healthkit" name in WebViewController.swift.
 *
 * All functions are no-ops when called outside the iOS native shell.
 */

/** Returns true when running in the iOS native shell with HealthKit support. */
export function isHealthKitAvailable(): boolean {
  return (
    "webkit" in window &&
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    "messageHandlers" in (window as any).webkit &&
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    "healthkit" in (window as any).webkit.messageHandlers
  );
}

/** Request HealthKit authorization from the native layer. */
export function requestHealthKitAuth(): void {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).webkit.messageHandlers.healthkit.postMessage({
    action: "requestAuth",
  });
}

/** Trigger a one-off HealthKit import of menstrual flow data into the engine. */
export function importFromHealthKit(): void {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).webkit.messageHandlers.healthkit.postMessage({
    action: "import",
  });
}
