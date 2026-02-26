import NiceModal from "@ebay/nice-modal-react";
import type { PaletteMode } from "@mui/material";
import type { AlertColor } from "@mui/material/Alert";
import Container from "@mui/material/Container";
import CssBaseline from "@mui/material/CssBaseline";
import { ThemeProvider } from "@mui/material/styles";
import useMediaQuery from "@mui/material/useMediaQuery";
import { AdapterDateFns } from "@mui/x-date-pickers/AdapterDateFns";
import { LocalizationProvider } from "@mui/x-date-pickers/LocalizationProvider";
import * as Sentry from "@sentry/react";
import type { QueryClient } from "@tanstack/react-query";
import { createRootRouteWithContext, HeadContent, Outlet, Scripts } from "@tanstack/react-router";
import { useCallback, useMemo, useState } from "react";
import { getAppInfo as apiGetAppInfo } from "../api/app.ts";
import type { AuthContextProps } from "../auth.tsx";
import { BackgroundImageProvider } from "../component/BackgroundImageProvider.tsx";
import { type Flash, Flashes } from "../component/Flashes.tsx";
import { Footer } from "../component/Footer.tsx";
import { LogoutHandler } from "../component/LogoutHandler.tsx";
import { NotificationsProvider } from "../component/NotificationsProvider.tsx";
import { OptionalQueueProvider } from "../component/OptionalQueueProvider.tsx";
import { QueueChat } from "../component/queue/QueueChat.tsx";
import { TopBar } from "../component/TopBar.tsx";
import { ColourModeContext } from "../contexts/ColourModeContext.tsx";
import { UserFlashCtx } from "../contexts/UserFlashCtx.tsx";
import { type ApiError, isApiError } from "../error.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import type { appInfoDetail } from "../schema/app.ts";
import { PermissionLevel } from "../schema/people.ts";
import { createThemeByMode } from "../theme.ts";
import { checkFeatureEnabled } from "../util/features.ts";
import { emptyOrNullString } from "../util/types.ts";

type RouterContext = {
	auth?: AuthContextProps;
	queryClient: QueryClient;
	appInfo: appInfoDetail;
};

export const Route = createRootRouteWithContext<RouterContext>()({
	component: Root,
	loader: async ({ context }) => {
		const appInfo = await context.queryClient.fetchQuery({ queryKey: ["appInfo"], queryFn: apiGetAppInfo });
		return { appInfo };
	},
	head: ({ loaderData }) => ({
		meta: [
			{ title: loaderData?.appInfo.site_name ?? "gbans" },
			{
				name: "description",
				content:
					loaderData?.appInfo.site_description ??
					"gbans is a web application for managing Team Fortress 2 communities",
			},
		],
	}),
});

function Root() {
	const initialTheme = (localStorage.getItem("theme") as PaletteMode) || "light";
	const { hasPermission } = useAuth();
	const [flashes, setFlashes] = useState<Flash[]>([]);
	const { appInfo } = Route.useLoaderData();
	const prefersDarkMode = useMediaQuery("(prefers-color-scheme: dark)");
	const [mode, setMode] = useState<"light" | "dark">(
		initialTheme ? initialTheme : prefersDarkMode ? "dark" : "light",
	);

	const updateMode = useCallback((prevMode: PaletteMode): PaletteMode => {
		const m = prevMode === "light" ? "dark" : ("light" as PaletteMode);
		localStorage.setItem("theme", m);
		return m;
	}, []);

	const colorMode = useMemo(
		() => ({
			toggleColorMode: () => {
				setMode(updateMode);
			},
		}),
		[updateMode],
	);

	const theme = useMemo(() => createThemeByMode(mode), [mode]);

	const sendFlash = useCallback(
		(level: AlertColor, message: string, heading = "", closable = true) => {
			if (flashes.length && flashes[flashes.length - 1]?.message === message) {
				// Skip duplicates
				return;
			}
			if (emptyOrNullString(heading)) {
				heading = level;
			}
			setFlashes([
				...flashes,
				{
					closable: closable ?? false,
					heading: heading,
					level: level,
					message: message,
				},
			]);
		},

		[flashes],
	);

	const sendError = useCallback(
		(error: unknown) => {
			if (!error) {
				return;
			}
			if (isApiError(error)) {
				sendFlash("error", (error as ApiError).detail, (error as ApiError).title, true);
			} else {
				sendFlash("error", (error as Error).message, (error as Error).name, true);
			}

			Sentry.captureException(error);
		},

		[sendFlash],
	);

	return (
		<UserFlashCtx.Provider value={{ flashes, setFlashes, sendFlash, sendError }}>
			<OptionalQueueProvider>
				<LocalizationProvider dateAdapter={AdapterDateFns}>
					<ColourModeContext.Provider value={colorMode}>
						<ThemeProvider theme={theme}>
							<Scripts />
							<BackgroundImageProvider />
							<NotificationsProvider>
								<NiceModal.Provider>
									<LogoutHandler />
									<CssBaseline />
									<HeadContent />
									<Container maxWidth={"lg"}>
										<TopBar appInfo={appInfo} />
										<div
											style={{
												marginTop: 24,
											}}
										>
											{checkFeatureEnabled("playerqueue_enabled") &&
												hasPermission(PermissionLevel.Moderator) && <QueueChat />}
											<Outlet />
										</div>
										<Footer appInfo={appInfo} />
									</Container>
									<Flashes />
								</NiceModal.Provider>
							</NotificationsProvider>
						</ThemeProvider>
					</ColourModeContext.Provider>
				</LocalizationProvider>
			</OptionalQueueProvider>
		</UserFlashCtx.Provider>
	);
}
