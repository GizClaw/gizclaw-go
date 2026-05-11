import { client as adminClient } from "@gizclaw/adminservice/client.gen";
import { client as publicClient } from "@gizclaw/serverpublic/client.gen";

export function configureAdminClients(): void {
  adminClient.setConfig({
    baseUrl: "/api/admin",
    responseStyle: "fields",
    throwOnError: false,
  });
  publicClient.setConfig({
    baseUrl: "/api/public",
    responseStyle: "fields",
    throwOnError: false,
  });
}
