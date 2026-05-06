// Create a new router instance
import type { QueryClient } from "@tanstack/react-query";
import { createRouter } from "@tanstack/react-router";
import { ErrorDetails } from "./component/ErrorDetails.tsx";
import { LoadingPlaceholder } from "./component/LoadingPlaceholder.tsx";
import { AppError, ErrorCode } from "./error.tsx";
import { routeTree } from "./routeTree.gen.ts";
import type { InfoResponse } from "./rpc/config/v1/config_pb.ts";
import { toTitleCase } from "./util/strings.ts";

export const newRouter = (queryClient: QueryClient, appInfo: InfoResponse) => {
	return createRouter({
		routeTree,
		defaultPreload: "intent",
		context: {
			queryClient,
			appInfo,
			title: (title?: string) => {
				return { title: title ? `${toTitleCase(title)} - ${appInfo.siteName}` : appInfo.siteName };
			},
		},
		defaultPendingComponent: LoadingPlaceholder,
		defaultErrorComponent: ({ error }) => {
			return <ErrorDetails error={new AppError(ErrorCode.Unknown, `${error}`)} />;
		},
		// Since we're using React Query, we don't want loader calls to ever be stale
		// This will ensure that the loader is always called when the route is preloaded or visited
		defaultPreloadStaleTime: 0,
		scrollRestoration: true,
	});
};
