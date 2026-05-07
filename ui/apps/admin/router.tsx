import { Navigate, Route, Routes } from "react-router-dom";

import { AdminLayout } from "./layout/AdminLayout";
import { VoiceDetailPage } from "./pages/ai/VoiceDetailPage";
import { VoicesListPage } from "./pages/ai/VoicesListPage";
import { WorkspaceTemplatesListPage } from "./pages/ai/WorkspaceTemplatesListPage";
import { WorkspacesListPage } from "./pages/ai/WorkspacesListPage";
import { ChannelDetailPage } from "./pages/firmware/ChannelDetailPage";
import { DepotDetailPage } from "./pages/firmware/DepotDetailPage";
import { FirmwareListPage } from "./pages/firmware/FirmwareListPage";
import { FirmwareUploadPage } from "./pages/firmware/FirmwareUploadPage";
import { PeerDetailPage } from "./pages/peers/PeerDetailPage";
import { PeersListPage } from "./pages/peers/PeersListPage";
import { OverviewPage } from "./pages/overview/OverviewPage";
import { CredentialsListPage } from "./pages/providers/CredentialsListPage";
import { MiniMaxTenantDetailPage } from "./pages/providers/MiniMaxTenantDetailPage";
import { MiniMaxTenantsListPage } from "./pages/providers/MiniMaxTenantsListPage";
import { VolcTenantDetailPage } from "./pages/providers/VolcTenantDetailPage";
import { VolcTenantsListPage } from "./pages/providers/VolcTenantsListPage";

export function AppRoutes(): JSX.Element {
  return (
    <Routes>
      <Route element={<AdminLayout />} path="/">
        <Route index element={<Navigate replace to="/overview" />} />
        <Route element={<OverviewPage />} path="overview" />
        <Route element={<PeersListPage />} path="peers" />
        <Route element={<PeerDetailPage />} path="peers/:publicKey" />
        <Route element={<FirmwareListPage />} path="firmware" />
        <Route element={<FirmwareUploadPage />} path="firmware/new" />
        <Route element={<DepotDetailPage />} path="firmware/:depot" />
        <Route element={<ChannelDetailPage />} path="firmware/:depot/:channel" />
        <Route element={<CredentialsListPage />} path="providers/credentials" />
        <Route element={<MiniMaxTenantsListPage />} path="providers/minimax-tenants" />
        <Route element={<MiniMaxTenantDetailPage />} path="providers/minimax-tenants/:name" />
        <Route element={<VolcTenantsListPage />} path="providers/volc-tenants" />
        <Route element={<VolcTenantDetailPage />} path="providers/volc-tenants/:name" />
        <Route element={<VoicesListPage />} path="ai/voices" />
        <Route element={<VoiceDetailPage />} path="ai/voices/:id" />
        <Route element={<WorkspaceTemplatesListPage />} path="ai/workspace-templates" />
        <Route element={<WorkspacesListPage />} path="ai/workspaces" />
      </Route>
      <Route element={<Navigate replace to="/overview" />} path="*" />
    </Routes>
  );
}
