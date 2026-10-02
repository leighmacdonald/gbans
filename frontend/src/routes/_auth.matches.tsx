import { createFileRoute } from "@tanstack/react-router";
import { Permission } from "../rpc/roles/v1/roles_pb.ts";
import { ensureFeatureEnabled } from "../util/features.ts";

export const Route = createFileRoute("/_auth/matches")({
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(
			(context.appInfo.statsEnabled && context.auth?.hasPermission(Permission.BAN_CREATE)) ?? false,
		);
	},
});
