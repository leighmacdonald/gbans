import { createFileRoute, redirect } from "@tanstack/react-router";
import { Permission } from "../rpc/roles/v1/roles_pb";

const mod_perms = [
	Permission.BAN_WRITE,
	Permission.ANTICHEAT_READ,
	Permission.BLOCKLIST_WRITE,
	Permission.WORDFILTER_WRITE,
	Permission.REPORT_ADMIN,
];
export const Route = createFileRoute("/_mod")({
	beforeLoad: ({ context, location }) => {
		// If the user is logged out, redirect them to the login page
		if (!context.auth?.isAuthenticated()) {
			throw redirect({
				to: "/login",
				search: {
					// Use the current location to power a redirect after login
					// (Do not use `router.state.resolvedLocation` as it can
					// potentially lag behind the actual current location)
					redirect: location.href,
				},
			});
		}
		const hasAtLeastoneModPerm = mod_perms.filter((p) => context.auth?.hasPermission(p)).length > 0;
		if (!hasAtLeastoneModPerm) {
			throw redirect({ to: "/permission" });
		}

		// Otherwise, return the user in context
		return context;
	},
});
