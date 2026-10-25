import { createClient } from "@connectrpc/connect";
import { createConnectQueryKey, useMutation, useQuery, useTransport } from "@connectrpc/connect-query";
import { useModal } from "@ebay/nice-modal-react";
import CableIcon from "@mui/icons-material/Cable";
import ConstructionIcon from "@mui/icons-material/Construction";
import DeleteIcon from "@mui/icons-material/Delete";
import ForumIcon from "@mui/icons-material/Forum";
import LoginIcon from "@mui/icons-material/Login";
import NotificationsActiveIcon from "@mui/icons-material/NotificationsActive";
import SettingsIcon from "@mui/icons-material/Settings";
import SettingsInputComponentIcon from "@mui/icons-material/SettingsInputComponent";
import SportsEsportsIcon from "@mui/icons-material/SportsEsports";
import Avatar from "@mui/material/Avatar";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import List from "@mui/material/List";
import ListItemText from "@mui/material/ListItemText";
import Stack from "@mui/material/Stack";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useCallback, useMemo, useState } from "react";
import { z } from "zod/v4";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { ConfirmationModal } from "../component/modal/ConfirmationModal.tsx";
import { SubHeading } from "../component/SubHeading.tsx";
import { TabButton } from "../component/TabButton.tsx";
import { TabSection } from "../component/TabSection.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { DiscordOAuthService } from "../rpc/discord/oauth/v1/discord_pb.ts";
import { profile as profileQuery } from "../rpc/discord/oauth/v1/discord-DiscordOAuthService_connectquery.ts";
import { PatreonService } from "../rpc/patreon/v1/patreon_pb.ts";
import type { ClientPref, Settings } from "../rpc/person/v1/person_pb.ts";
import { ClientPrefType } from "../rpc/person/v1/person_pb.ts";
import { editProfileSettings, profileSettings } from "../rpc/person/v1/person-PersonService_connectquery.ts";
import { Permission } from "../rpc/roles/v1/roles_pb.ts";
import { finalTransport } from "../transport.ts";
import { logErr } from "../util/errors.ts";
import { discordAvatarURL } from "../util/strings.ts";

const settingsSchema = z.object({
	section: z.enum(["general", "forums", "connections", "game"]).optional().default("general"),
});

export const Route = createFileRoute("/_auth/settings")({
	component: ProfileSettings,
	validateSearch: (search) => settingsSchema.parse(search),
	head: ({ match }) => ({
		meta: match.context.meta({ title: "User Settings", description: "Manage your account settings." }),
	}),
});

type userSettingTabs = "general" | "connections" | "forums" | "game";

function ProfileSettings() {
	const { sendFlash, sendError } = useUserFlashCtx();
	const { profile, hasPermission } = useAuth();
	const { appInfo } = Route.useRouteContext();
	const { section } = Route.useSearch();
	const [tab, setTab] = useState<userSettingTabs>(section);
	const navigate = useNavigate();

	const { data, isLoading } = useQuery(profileSettings);
	const queryClient = useQueryClient();
	const transport = useTransport();

	const mutation = useMutation(editProfileSettings, {
		onSuccess: () => {
			queryClient.invalidateQueries({
				queryKey: createConnectQueryKey({
					schema: profileSettings,
					cardinality: "finite",
					transport,
					input: {},
				}),
			});
			sendFlash("success", "Updated Settings");
		},
		onError: sendError,
	});

	const onTabClick = async (section: userSettingTabs) => {
		setTab(section);
		await navigate({ to: "/settings", replace: true, search: { section } });
	};

	const onSave = useCallback(
		async (settings: Settings, clientPrefs: Record<string, string>) => {
			return await mutation.mutateAsync({
				forumProfileMessages: settings.forumProfileMessages,
				forumSignature: settings.forumSignature,
				statsHidden: settings.statsHidden,
				matchNotifications: settings.matchNotifications,
				clientPrefs,
			});
		},
		[mutation.mutateAsync],
	);

	if (!data?.settings || isLoading) {
		return;
	}

	return (
		<ContainerWithHeader title={"User Settings"} iconLeft={<ConstructionIcon />} padding={2}>
			<Grid container spacing={2}>
				<Grid size={{ xs: 4, sm: 3, md: 2 }}>
					<Stack spacing={1}>
						<TabButton
							tab={"general"}
							onClick={onTabClick}
							icon={<SettingsIcon />}
							currentTab={tab}
							label={"General"}
						/>
						<TabButton
							tab={"game"}
							onClick={onTabClick}
							icon={<SportsEsportsIcon />}
							currentTab={tab}
							label={"Gameplay"}
						/>
						{hasPermission(Permission.FORUM_EDIT) && (
							<TabButton
								tab={"forums"}
								onClick={onTabClick}
								icon={<ForumIcon />}
								currentTab={tab}
								label={"Forums"}
							/>
						)}
						{(appInfo.patreonEnabled || appInfo.discordEnabled) && (
							<TabButton
								tab={"connections"}
								onClick={onTabClick}
								icon={<CableIcon />}
								currentTab={tab}
								label={"Connections"}
							/>
						)}
					</Stack>
				</Grid>
				<GeneralSection tab={tab} settings={data?.settings} onSave={onSave} />
				<GameplaySection
					tab={tab}
					settings={data?.settings}
					clientPrefsEnabled={data?.clientPrefsEnabled}
					clientPrefs={data?.clientPrefs}
					onSave={onSave}
				/>
				{hasPermission(Permission.FORUM_EDIT) && (
					<ForumSection tab={tab} settings={data?.settings} onSave={onSave} />
				)}
				<ConnectionsSection tab={tab} settings={data?.settings} onSave={onSave} patreonId={profile.patreonId} />
			</Grid>
		</ContainerWithHeader>
	);
}

type GeneralProps = {
	statsHidden: boolean;
	matchNotifications: boolean;
};

const GeneralSection = ({
	tab,
	settings,
	onSave,
}: {
	tab: userSettingTabs;
	settings: Settings;
	onSave: (s: Settings, prefs: Record<string, string>) => void;
}) => {
	const [notifPerms, setNotifPerms] = useState(Notification.permission);

	const notificationsSupported = useMemo(() => {
		return "Notification" in window;
	}, []);

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			onSave({ ...settings, ...value }, {});
		},
		defaultValues: {
			statsHidden: settings.statsHidden,
			matchNotifications: settings.matchNotifications,
		} as GeneralProps,
	});

	const togglePerms = async () => {
		setNotifPerms(await Notification.requestPermission());
	};

	return (
		<TabSection
			tab={"general"}
			currentTab={tab}
			label={"General"}
			description={"Core settings that don't fit into a subcategory"}
		>
			<form
				onSubmit={async (e) => {
					e.preventDefault();
					e.stopPropagation();
					await form.handleSubmit();
				}}
			>
				<Grid container spacing={2}>
					{notificationsSupported && (
						<Grid size={{ xs: 12 }}>
							<div>
								<Grid container spacing={1}>
									<Grid size={{ xs: 12, md: 6 }}>
										<Stack spacing={1}>
											<Typography>Show desktop notifications?</Typography>
											{notifPerms !== "granted" ? (
												<Button
													sx={{ width: 300 }}
													variant={"contained"}
													color={"success"}
													onClick={togglePerms}
													startIcon={<NotificationsActiveIcon />}
												>
													Enable Desktop Notifications
												</Button>
											) : (
												<Tooltip
													title={
														"Please see the links to the right for instructions on how to disable"
													}
												>
													<span>
														<Button
															sx={{ width: 300 }}
															disabled={true}
															variant={"contained"}
															color={"success"}
															onClick={togglePerms}
															startIcon={<NotificationsActiveIcon />}
														>
															Desktop Notifications Enabled
														</Button>
													</span>
												</Tooltip>
											)}
										</Stack>
									</Grid>
									<Grid size={{ xs: 12, md: 6 }}>
										<Typography>How to disable notifications: </Typography>
										<List>
											<ListItemText>
												<Link
													href={
														"https://support.google.com/chrome/answer/114662?sjid=2540186662959230327-NC&visit_id=638611827496769425-656746251&rd=1"
													}
												>
													Google Chrome
												</Link>
											</ListItemText>
											<ListItemText>
												<Link
													href={
														"https://support.mozilla.org/en-US/kb/push-notifications-firefox"
													}
												>
													Mozilla Firefox
												</Link>
											</ListItemText>
										</List>
									</Grid>
								</Grid>
							</div>
						</Grid>
					)}
					<Grid size={{ xs: 12 }}>
						<form.AppField
							name={"statsHidden"}
							validators={{
								onChange: z.boolean(),
							}}
							children={(field) => {
								return <field.CheckboxField label={"Hide personal stats on profile"} />;
							}}
						/>
						<SubHeading>It is still viewable by yourself.</SubHeading>
					</Grid>

					<Grid size={{ xs: 12 }}>
						<form.AppField
							name={"matchNotifications"}
							validators={{
								onChange: z.boolean(),
							}}
							children={(field) => {
								return (
									<field.CheckboxField label={"Notify me in-game when my match results are ready"} />
								);
							}}
						/>
						<SubHeading>
							Sends a private message with a link to the match after it has been imported. Only sent while
							you are still connected to one of the servers.
						</SubHeading>
					</Grid>

					<Grid size={{ xs: 12 }}>
						<form.AppForm>
							<ButtonGroup>
								<form.SubmitButton />
							</ButtonGroup>
						</form.AppForm>
					</Grid>
				</Grid>
			</form>
		</TabSection>
	);
};

// Clientpref values live in the sourcemod cookie cache as strings: booleans
// are "1"/"0" and an empty value removes the override so the server default
// applies.
const boolWireToForm = (value: string) => (value === "1" ? "on" : value === "0" ? "off" : "default");
const boolFormToWire = (value: string) => (value === "on" ? "1" : value === "off" ? "0" : "");

const clientPrefLabel = (pref: ClientPref) => {
	if (pref.description) {
		return pref.description;
	}

	return pref.name.replaceAll("_", " ").replace(/^\w/, (c) => c.toUpperCase());
};

const clientPrefHint = (pref: ClientPref) => {
	if (!pref.editable) {
		return "This setting is read-only.";
	}

	if (pref.type === ClientPrefType.BOOLEAN) {
		return "Default leaves the choice to the server.";
	}

	return "Leave empty to use the server default.";
};

const clientPrefValidator = (pref: ClientPref) => {
	if (pref.type === ClientPrefType.BOOLEAN) {
		return z.enum(["on", "default", "off"]);
	}

	if (pref.type === ClientPrefType.NUMBER) {
		return z.string().refine((v) => v === "" || /^\d+$/.test(v), "Enter a whole number");
	}

	return z.string().max(100);
};

const GameplaySection = ({
	tab,
	settings,
	clientPrefsEnabled,
	clientPrefs,
	onSave,
}: {
	tab: userSettingTabs;
	settings: Settings;
	clientPrefsEnabled: boolean;
	clientPrefs: ClientPref[];
	onSave: (s: Settings, prefs: Record<string, string>) => void;
}) => {
	const form = useAppForm({
		onSubmit: async ({ value }) => {
			const prefs: Record<string, string> = {};
			for (const pref of clientPrefs) {
				const v = value[pref.name] ?? "";
				prefs[pref.name] = pref.type === ClientPrefType.BOOLEAN ? boolFormToWire(v) : v;
			}
			onSave(settings, prefs);
		},
		defaultValues: Object.fromEntries(
			clientPrefs.map((p) => [p.name, p.type === ClientPrefType.BOOLEAN ? boolWireToForm(p.value) : p.value]),
		),
	});

	if (!clientPrefsEnabled) {
		return (
			<TabSection
				tab={"game"}
				currentTab={tab}
				label={"Gameplay"}
				description={"Configure your in-game clientprefs"}
			>
				<Typography>In-game clientprefs are disabled on this server.</Typography>
			</TabSection>
		);
	}

	return (
		<TabSection tab={"game"} currentTab={tab} label={"Gameplay"} description={"Configure your in-game clientprefs"}>
			<form
				onSubmit={async (e) => {
					e.preventDefault();
					e.stopPropagation();
					await form.handleSubmit();
				}}
			>
				<Grid container spacing={2}>
					{clientPrefs.map((pref) => {
						return (
							<Grid key={pref.name} size={{ xs: 12 }}>
								<form.AppField
									name={pref.name}
									validators={{
										onChange: clientPrefValidator(pref),
									}}
									children={(field) => {
										if (pref.type === ClientPrefType.BOOLEAN) {
											return (
												<field.TriStateField
													label={clientPrefLabel(pref)}
													disabled={!pref.editable}
												/>
											);
										}

										return (
											<field.TextField
												label={clientPrefLabel(pref)}
												disabled={!pref.editable}
												maxLength={100}
											/>
										);
									}}
								/>
								<SubHeading>{clientPrefHint(pref)}</SubHeading>
							</Grid>
						);
					})}

					<Grid size={{ xs: 12 }}>
						<form.AppForm>
							<ButtonGroup>
								<form.ResetButton />
								<form.SubmitButton />
							</ButtonGroup>
						</form.AppForm>
					</Grid>
				</Grid>
			</form>
		</TabSection>
	);
};

const ForumSection = ({
	tab,
	settings,
	onSave,
}: {
	tab: userSettingTabs;
	settings: Settings;
	onSave: (s: Settings, prefs: Record<string, string>) => void;
}) => {
	const form = useAppForm({
		onSubmit: async ({ value }) => {
			onSave({ ...settings, ...value }, {});
		},
		defaultValues: {
			forumSignature: settings.forumSignature,
			forumProfileMessages: settings.forumProfileMessages,
		},
	});

	return (
		<TabSection tab={"forums"} currentTab={tab} label={"Forums"} description={"Configure forum features"}>
			<form
				onSubmit={async (e) => {
					e.preventDefault();
					e.stopPropagation();
					await form.handleSubmit();
				}}
			>
				<Grid container spacing={2}>
					<Grid size={{ xs: 12 }}>
						<form.AppField
							name={"forumSignature"}
							validators={{
								onChange: z.string(),
							}}
							children={(field) => {
								return (
									<field.MarkdownField
										{...field}
										value={field.state.value}
										label={"Your forum signature"}
										rows={10}
									/>
								);
							}}
						/>
						<SubHeading>It is still viewable by yourself.</SubHeading>
					</Grid>

					<Grid size={{ xs: 12 }}>
						<form.AppField
							name={"forumProfileMessages"}
							validators={{
								onChange: z.boolean(),
							}}
							children={(field) => {
								return <field.CheckboxField label={"Enable people to sign your profile."} />;
							}}
						/>
						<SubHeading>It is still viewable by yourself.</SubHeading>
					</Grid>

					<Grid size={{ xs: 12 }}>
						<form.AppForm>
							<ButtonGroup>
								<form.ResetButton />
								<form.SubmitButton />
							</ButtonGroup>
						</form.AppForm>
					</Grid>
				</Grid>
			</form>
		</TabSection>
	);
};

const ConnectionsSection = ({
	tab,
	patreonId,
}: {
	tab: userSettingTabs;
	settings: Settings;
	onSave: (s: Settings, prefs: Record<string, string>) => void;
	patreonId: string;
}) => {
	const queryClient = useQueryClient();
	const { profile } = useAuth();
	const { sendFlash } = useUserFlashCtx();
	const confirmModal = useModal(ConfirmationModal);
	const { appInfo } = Route.useRouteContext();

	const { data: user, isLoading } = useQuery(profileQuery);

	const followPatreonCallback = async () => {
		const client = createClient(PatreonService, finalTransport);
		const resp = await queryClient.fetchQuery({
			queryKey: createConnectQueryKey({
				schema: PatreonService,
				transport: finalTransport,
				cardinality: "finite",
			}),
			queryFn: async () => {
				return await client.patronLogin({});
			},
		});

		window.open(resp.url, "_self");
	};

	const followDiscordCallback = async () => {
		const discordClient = createClient(DiscordOAuthService, finalTransport);
		const resp = await queryClient.fetchQuery({
			queryKey: createConnectQueryKey({
				schema: DiscordOAuthService,
				transport: finalTransport,
				cardinality: "finite",
			}),
			queryFn: async () => {
				return await discordClient.login({});
			},
		});

		window.open(resp.loginUrl, "_self");
	};

	const onForgetDiscord = async () => {
		const confirmed = await confirmModal.show({
			title: "Are you sure you want to remove discord connection?",
			children: "You will need to reconnect if you want to use related features again in the future.",
		});
		if (!confirmed) {
			return;
		}
		try {
			const discordClient = createClient(DiscordOAuthService, finalTransport);
			await queryClient.fetchQuery({
				queryKey: createConnectQueryKey({
					schema: DiscordOAuthService,
					transport: finalTransport,
					cardinality: "finite",
				}),
				queryFn: async () => {
					return await discordClient.logout({});
				},
			});

			queryClient.setQueryData(["discordProfile", { steamID: profile.steamId }], {});

			sendFlash("success", "Logged out successfully");
		} catch (e) {
			logErr(e);
			sendFlash("error", "Could not logout fully");
		}
	};

	const onForget = async () => {
		const confirmed = await confirmModal.show({
			title: "Are you sure you want to remove patreon connection?",
			children: "You will need to reconnect if you want to use related features again in the future.",
		});
		if (!confirmed) {
			return;
		}
		try {
			const client = createClient(PatreonService, finalTransport);
			await queryClient.fetchQuery({
				queryKey: createConnectQueryKey({
					schema: PatreonService,
					transport: finalTransport,
					cardinality: "finite",
				}),
				queryFn: async () => {
					return await client.patronLogout({});
				},
			});

			sendFlash("success", "Logged out successfully");
		} catch (e) {
			logErr(e);
			sendFlash("error", "Could not log out fully");
		}
	};

	return (
		<TabSection
			tab={"connections"}
			currentTab={tab}
			label={"Connections"}
			description={"Configure your 3rd party connections to us."}
		>
			<Grid container spacing={2}>
				{appInfo.patreonEnabled ? (
					patreonId ? (
						<Grid size={{ xs: 12 }}>
							<Typography variant={"h3"}>Patreon</Typography>
							<Box>
								<SubHeading>
									You are currently authenticated to us as:{" "}
									<Link href={`https://www.patreon.com/user/creators?u=${patreonId}`}>
										{patreonId}
									</Link>
								</SubHeading>
							</Box>
							<Button
								color={"error"}
								startIcon={<DeleteIcon />}
								variant={"contained"}
								onClick={onForget}
								fullWidth={false}
							>
								Disconnect Patreon
							</Button>
						</Grid>
					) : (
						<Grid size={{ xs: 12 }}>
							<Button
								variant={"contained"}
								color={"success"}
								onClick={followPatreonCallback}
								startIcon={<SettingsInputComponentIcon />}
							>
								Connect Patreon
							</Button>
						</Grid>
					)
				) : null}
				{appInfo.discordEnabled ? (
					!isLoading && user?.discordProfile?.discriminator ? (
						<Grid size={{ xs: 12 }}>
							<Typography>You are connected to us as: {user.discordProfile.discriminator}</Typography>
							<Button
								variant={"contained"}
								color={"error"}
								onClick={onForgetDiscord}
								startIcon={
									<Avatar
										src={discordAvatarURL(user.discordProfile)}
										sx={{ height: 20, width: 20 }}
									/>
								}
							>
								Disconnect Discord
							</Button>
						</Grid>
					) : (
						<Grid size={{ xs: 12 }}>
							<Button
								variant={"contained"}
								color={"success"}
								onClick={followDiscordCallback}
								startIcon={<LoginIcon />}
							>
								Connect Discord
							</Button>
						</Grid>
					)
				) : null}
			</Grid>
		</TabSection>
	);
};
