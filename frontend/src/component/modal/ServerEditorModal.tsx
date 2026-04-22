import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import RouterIcon from "@mui/icons-material/Router";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import { useAppForm } from "../../contexts/formContext.tsx";
import { randomStringAlphaNum } from "../../util/strings.ts";
import { Heading } from "../Heading";
import type { Server } from "../../rpc/servers/v1/servers_pb.ts";
import { useMutation } from "@connectrpc/connect-query";
import { editServer } from "../../rpc/servers/v1/servers-ServersService_connectquery.ts";

// const schema = z.object({
// 	shortName: z
// 		.string()
// 		.min(3)
// 		.regex(/\w{3,}/),
// 	name: z.string().min(1),
// 	address: z.string().min(1),
// 	port: z.number().min(1024).max(65535),
// 	password: z.string().length(20),
// 	rcon: z.string().min(4),
// 	region: z.string().min(1),
// 	cc: z.string().length(2),
// 	latitude: z.number().min(-90).max(99),
// 	longitude: z.number().min(-180).max(180),
//     isEnabled: z.boolean(),
//     enableStats: z.boolean(),
//     logSecret: z.number().min(100000).max(999999999),
//     addressInternal: z.string(),
//     sdrEnabled: z.boolean(),
//     discordSeedRoleIds: z.string().array(),
// });

export const ServerEditorModal = NiceModal.create(({ server }: { server?: Server }) => {
	const modal = useModal();
	const defaultValues = {
		shortName: server?.shortName ?? "",
		name: server?.name ?? "",
		address: server?.address ?? "",
		port: server?.port ?? 27015,
		password: server?.password ?? randomStringAlphaNum(20),
		rcon: server?.rcon ?? "",
		region: server?.region ?? "",
		cc: server?.cc ?? "",
		latitude: server?.latLong?.latitude ?? 0,
		longitude: server?.latLong?.longitude ?? 0,
		isEnabled: server?.isEnabled ?? true,
		enableStats: server?.enableStats ?? true,
		logSecret: server?.logSecret ?? Math.floor(Math.random() * 89999999 + 10000000),
		addressInternal: server?.addressInternal ?? "",
		sdrEnabled: server?.sdrEnabled ?? false,
		discordSeedRoleIds: server?.discordSeedRoleIds.join(",") ?? "",
	};

	const mutation = useMutation(editServer, {});
	// 	mutationKey: ["adminServer"],
	// 	mutationFn: async (values: ServerEditValues) => {
	// 		const opts: SaveServerOpts = {
	// 			short_name: values.short_name,
	// 			name: values.name,
	// 			address: values.address,
	// 			port: values.port,
	// 			password: values.password,
	// 			rcon: values.rcon,
	// 			region: values.region,
	// 			cc: values.cc,
	// 			latitude: values.latitude,
	// 			longitude: values.longitude,
	// 			reserved_slots: values.reserved_slots,
	// 			is_enabled: values.is_enabled,
	// 			enable_stats: values.enabled_stats,
	// 			log_secret: values.log_secret,
	// 			address_internal: values.address_internal,
	// 			sdr_enabled: values.sdr_enabled,
	// 			discord_seed_role_ids: values.discord_seed_role_ids.split(","),
	// 		};
	// 		const ac = new AbortController();
	// 		if (server?.server_id) {
	// 			modal.resolve(await apiSaveServer(server.server_id, opts, ac.signal));
	// 		} else {
	// 			modal.resolve(await apiCreateServer(opts, ac.signal));
	// 		}
	// 		await modal.hide();
	// 	},
	// });

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate({ server: { ...value, discordSeedRoleIds: value.discordSeedRoleIds.split(",") } });
		},
		defaultValues,
		// validators: {
		// 	onSubmit: schema,
		// },
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
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"rcon"}
								children={(field) => {
									return <field.TextField label={"RCON Password"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
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

						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"discordSeedRoleIds"}
								children={(field) => {
									return <field.TextField label={"Discord Seed Channel(s) (comma seperated)"} />;
								}}
							/>
						</Grid>
					</Grid>
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
