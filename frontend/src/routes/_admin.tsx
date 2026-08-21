import { createFileRoute, redirect } from "@tanstack/react-router";
import { Permission } from "../rpc/roles/v1/roles_pb";

export const Route = createFileRoute("/_admin")({
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

		if (!context.auth?.hasPermission(Permission.BAN_WRITE)) {
			throw redirect({ to: "/permission" });
		}

		// Otherwise, return the user in context
		return context;
	},
});
