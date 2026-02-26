// Create a new router instance
import type { QueryClient } from "@tanstack/react-query";
import { createRouter } from "@tanstack/react-router";
import { ErrorDetails } from "./component/ErrorDetails.tsx";
import { LoadingPlaceholder } from "./component/LoadingPlaceholder.tsx";
import { AppError, ErrorCode } from "./error.tsx";
import { routeTree } from "./routeTree.gen.ts";

export const newRouter = (queryClient: QueryClient) => {
	return createRouter({
		routeTree,
		defaultPreload: "intent",
		context: {
			queryClient,
		},
		defaultPendingComponent: LoadingPlaceholder,
		defaultErrorComponent: () => {
			return <ErrorDetails error={new AppError(ErrorCode.Unknown, "Unexpected error")} />;
		},
		// Since we're using React Query, we don't want loader calls to ever be stale
		// This will ensure that the loader is always called when the route is preloaded or visited
		defaultPreloadStaleTime: 0,
		scrollRestoration: true,
	});
};
