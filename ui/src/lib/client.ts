import { createClient, type Interceptor } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { CycleTrackerService } from "@gen/openmenses/v1/service_pb";

// When running inside a native shell, the engine injects its config into
// window.__OPENMENSES_ENGINE__. In dev mode this is undefined and we fall
// back to the Vite proxy at window.location.origin.
const engineConfig = window.__OPENMENSES_ENGINE__;

const baseUrl = engineConfig
  ? `http://127.0.0.1:${engineConfig.port}`
  : window.location.origin;

const interceptors: Interceptor[] = [];

if (engineConfig?.authToken) {
  interceptors.push((next) => async (req) => {
    req.header.set("Authorization", `Bearer ${engineConfig.authToken}`);
    return next(req);
  });
}

const transport = createConnectTransport({ baseUrl, interceptors });

export const client = createClient(CycleTrackerService, transport);

export const DEFAULT_PARENT = "users/default";
