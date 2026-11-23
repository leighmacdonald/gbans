import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import RouterIcon from "@mui/icons-material/Router";
import Alert from "@mui/material/Alert";
import ButtonGroup from "@mui/material/ButtonGroup";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { useState } from "react";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { seedRoleIDs } from "../../rpc/discord/v1/discord-DiscordService_connectquery.ts";
import { EditServerRequestSchema, type Server, ServerSchema } from "../../rpc/servers/v1/servers_pb.ts";
import { editServer } from "../../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { buckets } from "../../rpc/stats/v1/stats-StatsService_connectquery.ts";
import { logErr } from "../../util/errors.ts";
import { randomNum, randomStringAlphaNum } from "../../util/strings.ts";
import { Heading } from "../Heading";
import { LoadingPlaceholder } from "../LoadingPlaceholder.tsx";

// Mirrors the buf.validate rules in proto/servers/v1/servers.proto so bad
// values are caught before the RPC is sent.
const hostPattern = /^[a-zA-Z0-9][a-zA-Z0-9.-]*$/;
const validHost = (value: string) => value === "" || hostPattern.test(value);

const serverSchema = z.object({
	shortName: z.string().min(1, "Short name is required").max(32, "Maximum 32 characters"),
	name: z.string().min(1, "Name is required"),
	address: z.string().min(1, "Address is required").refine(validHost, "Enter a valid IP or hostname"),
	addressInternal: z.string().refine(validHost, "Enter a valid IP or hostname"),
	port: z.number().int().min(1024, "Minimum port is 1024").max(65535, "Maximum port is 65535"),
	password: z.string().min(1, "Server auth key is required"),
	stvPassword: z.string(),
	rcon: z.string().min(1, "RCON password is required"),
	region: z.string().min(1, "Region is required"),
	cc: z.string().length(2, "Use a 2-letter country code"),
	latitude: z.number().min(-90, "Latitude must be between -90 and 90").max(90, "Latitude must be between -90 and 90"),
	longitude: z
		.number()
		.min(-180, "Longitude must be between -180 and 180")
		.max(180, "Longitude must be between -180 and 180"),
	isEnabled: z.boolean(),
	enableStats: z.boolean(),
	logSecret: z.number().int().min(100001, "Log secret must be greater than 100000"),
	sdrEnabled: z.boolean(),
	discordSeedChannelID: z.string(),
	discordSeedRoleIds: z.array(z.string()),
	statsBucketId: z.number().int().optional(),
});

export const ServerEditorModal = NiceModal.create(({ server }: { server?: Server }) => {
	const { data: roles, isLoading: isLoadingRoles } = useQuery(seedRoleIDs);
	const { data, isLoading: isLoadingBuckets } = useQuery(buckets);

	const modal = useModal();
	const [submitError, setSubmitError] = useState<string | null>(null);
	const defaultValues: z.input<typeof serverSchema> = {
		shortName: server?.shortName ?? "",
		name: server?.name ?? "",
		address: server?.address ?? "",
		port: server?.port ?? 27015,
		password: server?.password ?? randomStringAlphaNum(20),
		stvPassword: server?.stvPassword ?? "",
		rcon: server?.rcon ?? "",
		region: server?.region ?? "",
		cc: server?.cc ?? "",
		latitude: server?.latLong?.latitude ?? 0,
		longitude: server?.latLong?.longitude ?? 0,
		isEnabled: server?.isEnabled ?? true,
		enableStats: server?.enableStats ?? true,
		logSecret: server?.logSecret ?? Math.max(100_001, randomNum()),
		addressInternal: server?.addressInternal ?? "",
		sdrEnabled: server?.sdrEnabled ?? false,
		discordSeedChannelID: server?.discordSeedChannelId ?? "",
		discordSeedRoleIds: server?.discordSeedRoleIds ?? [],
		statsBucketId: server?.statsBucketId,
	};

	const mutation = useMutation(editServer, {
		onSuccess: (response) => {
			modal.resolve(response.server);
			modal.hide();
		},
		onError: (error) => {
			logErr(error);
			setSubmitError(error instanceof Error ? error.message : "Failed to save server");
		},
	});

	const form = useAppForm({
		formId: server ? `server-${server.serverId}` : "server-create",
		onSubmit: async ({ value }) => {
			setSubmitError(null);
			mutation.mutate(
				create(EditServerRequestSchema, {
					server: create(ServerSchema, {
						...value,
						latLong: {
							latitude: value.latitude,
							longitude: value.longitude,
						},
						serverId: server?.serverId,
					}),
				}),
			);
		},
		defaultValues,
		validators: {
			onBlur: serverSchema,
			onSubmit: serverSchema,
		},
	});

	return (
		<Dialog fullWidth {...muiDialogV5(modal)}>
			<form
				onSubmit={async (e) => {
					e.preventDefault();
					e.stopPropagation();
					await form.handleSubmit();
				}}
			>
				<DialogTitle component={Heading} iconLeft={<RouterIcon />}>
					{server?.serverId ? "Edit" : "Create"} Server
				</DialogTitle>

				<DialogContent>
					{submitError && (
						<Alert severity={"error"} sx={{ mb: 2 }} onClose={() => setSubmitError(null)}>
							{submitError}
						</Alert>
					)}
					{isLoadingRoles || isLoadingBuckets ? (
						<LoadingPlaceholder />
					) : (
						<Grid container spacing={2}>
							<Grid size={{ xs: 4 }}>
								<form.AppField
									name={"shortName"}
									children={(field) => {
										return (
											<field.TextField
												label={"Short Name/Tag"}
												helperText={"A short, unique, identifier."}
											/>
										);
									}}
								/>
							</Grid>
							<Grid size={{ xs: 4 }}>
								<form.AppField
									name={"name"}
									children={(field) => {
										return <field.TextField label={"Long Name"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 4 }}>
								<form.AppField
									name={"address"}
									children={(field) => {
										return <field.TextField label={"Address"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 4 }}>
								<form.AppField
									name={"port"}
									children={(field) => {
										return <field.NumberField label={"Port"} min={1024} max={65535} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 8 }}>
								<form.AppField
									name={"addressInternal"}
									children={(field) => {
										return (
											<field.TextField
												label={"Address Internal"}
												helperText={
													"A private network/VPN to access the host. Used for SSH. If empty the normal address is used."
												}
											/>
										);
									}}
								/>
							</Grid>
							<Grid size={{ xs: 8 }}>
								<form.AppField
									name={"sdrEnabled"}
									children={(field) => {
										return <field.CheckboxField label={"Enable SDR Support"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 4 }}>
								<form.AppField
									name={"password"}
									children={(field) => {
										return <field.TextField label={"Server Auth Key"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 4 }}>
								<form.AppField
									name={"rcon"}
									children={(field) => {
										return <field.TextField label={"RCON Password"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 4 }}>
								<form.AppField
									name={"stvPassword"}
									children={(field) => {
										return (
											<field.TextField
												label={"SourceTV Password"}
												helperText={"Optional password for the SourceTV broadcast."}
											/>
										);
									}}
								/>
							</Grid>
							<Grid size={{ xs: 4 }}>
								<form.AppField
									name={"logSecret"}
									children={(field) => {
										return <field.NumberField label={"Log Secret"} min={0} max={999999999} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6 }}>
								<form.AppField
									name={"region"}
									children={(field) => {
										return <field.TextField label={"Region"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6 }}>
								<form.AppField
									name={"cc"}
									children={(field) => {
										return <field.TextField label={"Country Code"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6 }}>
								<form.AppField
									name={"latitude"}
									children={(field) => {
										return <field.NumberField label={"Latitude"} min={-90} max={90} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6 }}>
								<form.AppField
									name={"longitude"}
									children={(field) => {
										return <field.NumberField label={"Longitude"} min={-180} max={180} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 4 }}>
								<form.AppField
									name={"isEnabled"}
									children={(field) => {
										return <field.CheckboxField label={"Is Enabled"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 4 }}>
								<form.AppField
									name={"enableStats"}
									children={(field) => {
										return <field.CheckboxField label={"Stats Enabled"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 4 }}>
								<form.AppField
									name={"statsBucketId"}
									children={(field) => {
										return (
											<field.SelectBucketField
												label={"Stat Bucket"}
												items={data?.buckets ?? []}
												renderItem={(i) => {
													return <MenuItem value={i.statsBucketId}>{i.bucketName}</MenuItem>;
												}}
											/>
										);
									}}
								/>
							</Grid>

							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"discordSeedChannelID"}
									children={(field) => {
										return <field.TextField label={"Discord Seed Channel ID"} />;
									}}
								/>
							</Grid>

							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"discordSeedRoleIds"}
									children={(field) => {
										return (
											<field.SelectDiscordRolesField
												items={roles?.roles ?? []}
												multiple={true}
												label={"Discord Seed Roles(s)"}
												renderItem={(i) => {
													return (
														<MenuItem value={i.id} key={i.id}>
															{i.name}
														</MenuItem>
													);
												}}
											/>
										);
									}}
								/>
							</Grid>
						</Grid>
					)}
				</DialogContent>
				<DialogActions>
					<Grid container>
						<Grid size={{ xs: 12 }}>
							<form.AppForm>
								<ButtonGroup>
									<form.CloseButton />
									<form.ResetButton />
									<form.SubmitButton />
								</ButtonGroup>
							</form.AppForm>
						</Grid>
					</Grid>
				</DialogActions>
			</form>
		</Dialog>
	);
});
