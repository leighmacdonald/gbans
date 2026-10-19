import { createFileRoute } from "@tanstack/react-router";
import { ErrorDetails } from "../component/ErrorDetails.tsx";
import { AppError, ErrorCode } from "../error.tsx";

export const Route = createFileRoute("/_auth/permission")({
	component: () => {
		const err = new AppError(ErrorCode.PermissionDenied);
		return <ErrorDetails error={err} />;
	},
	head: ({ match }) => ({
		meta: match.context.meta({
			title: "Permission Denied",
			description: "You do not have permission to view this page.",
		}),
	}),
});
