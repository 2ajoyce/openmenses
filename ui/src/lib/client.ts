import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { CycleTrackerService } from "@gen/openmenses/v1/service_pb";

const transport = createConnectTransport({
  baseUrl: window.location.origin,
});

export const client = createClient(CycleTrackerService, transport);

export const DEFAULT_PARENT = "users/default";
