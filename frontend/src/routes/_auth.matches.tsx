import { createFileRoute } from "@tanstack/react-router";
import { Privilege } from "../rpc/person/v1/privilege_pb.ts";
import { ensureFeatureEnabled } from "../util/features.ts";

export const Route = createFileRoute("/_auth/matches")({
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(
			(context.appInfo.statsEnabled && context.auth?.hasPermission(Privilege.MODERATOR)) ?? false,
		);
	},
});
