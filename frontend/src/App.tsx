import { TransportProvider } from "@connectrpc/connect-query";
import { type QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { type AnyRouter, RouterProvider } from "@tanstack/react-router";
import { StrictMode } from "react";
import { AuthProvider } from "./auth.tsx";
import { useAuth } from "./hooks/useAuth.ts";
import { finalTransport } from "./transport.ts";

export function App({ queryClient, router }: { queryClient: QueryClient; router: AnyRouter }) {
	return (
		<TransportProvider transport={finalTransport}>
			<QueryClientProvider client={queryClient}>
				<AuthProvider>
					<StrictMode>
						<InnerApp router={router} />
					</StrictMode>
				</AuthProvider>
			</QueryClientProvider>
		</TransportProvider>
	);
}

const InnerApp = ({ router }: { router: AnyRouter }) => {
	const auth = useAuth();

	return <RouterProvider defaultPreload={"intent"} router={router} context={{ auth }} />;
};
