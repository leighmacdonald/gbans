import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import RouterIcon from "@mui/icons-material/Router";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import { useMutation } from "@tanstack/react-query";
import { z } from "zod/v4";
import { apiCreateServer, apiSaveServer } from "../../api";
import { useAppForm } from "../../contexts/formContext.tsx";
import type { SaveServerOpts, Server } from "../../schema/server.ts";
import { randomStringAlphaNum } from "../../util/strings.ts";
import { Heading } from "../Heading";

type ServerEditValues = {
	short_name: string;
	name: string;
	address: string;
	address_internal: string;
	sdr_enabled: boolean;
	port: number;
	password: string;
	rcon: string;
	region: string;
	cc: string;
	latitude: number;
	longitude: number;
	reserved_slots: number;
	is_enabled: boolean;
	enabled_stats: boolean;
	log_secret: number;
	discord_seed_role_ids: string;
};

const schema = z.object({
	short_name: z
		.string()
		.min(3)
		.regex(/\w{3,}/),
	name: z.string().min(1),
	address: z.string().min(1),
	port: z.number().min(1024).max(65535),
	password: z.string().length(20),
	rcon: z.string().min(6),
	region: z.string().min(1),
	cc: z.string().length(2),
	latitude: z.number().min(-90).max(99),
	longitude: z.number().min(-180).max(180),
	reserved_slots: z.number().min(0).max(100),
	is_enabled: z.boolean(),
	enabled_stats: z.boolean(),
	log_secret: z.coerce.number().min(100000).max(999999999),
	address_internal: z.string(),
	sdr_enabled: z.boolean(),
	discord_seed_role_ids: z.string(),
});

export const ServerEditorModal = NiceModal.create(({ server }: { server?: Server }) => {
	const modal = useModal();
	const defaultValues: z.input<typeof schema> = {
		short_name: server?.short_name ?? "",
		name: server?.name ?? "",
		address: server?.address ?? "",
		port: server?.port ?? 27015,
		password: server?.password ?? randomStringAlphaNum(20),
		rcon: server?.rcon ?? "",
		region: server?.region ?? "",
		cc: server?.cc ?? "",
		latitude: server?.latitude ?? 0,
		longitude: server?.longitude ?? 0,
		reserved_slots: server?.reserved_slots ?? 0,
		is_enabled: server?.is_enabled ?? true,
		enabled_stats: server?.enable_stats ?? true,
		log_secret: server?.log_secret ?? Math.floor(Math.random() * 89999999 + 10000000),
		address_internal: server?.address_internal ?? "",
		sdr_enabled: server?.sdr_enabled ?? false,
		discord_seed_role_ids: server?.discord_seed_role_ids.join(",") ?? "",
	};

	const mutation = useMutation({
		mutationKey: ["adminServer"],
		mutationFn: async (values: ServerEditValues) => {
			const opts: SaveServerOpts = {
				short_name: values.short_name,
				name: values.name,
				address: values.address,
				port: values.port,
				password: values.password,
				rcon: values.rcon,
				region: values.region,
				cc: values.cc,
				lat: values.latitude,
				lon: values.longitude,
				reserved_slots: values.reserved_slots,
				is_enabled: values.is_enabled,
				enable_stats: values.enabled_stats,
				log_secret: values.log_secret,
				address_internal: values.address_internal,
				sdr_enabled: values.sdr_enabled,
				discord_seed_role_ids: values.discord_seed_role_ids.split(","),
			};
			if (server?.server_id) {
				modal.resolve(await apiSaveServer(server.server_id, opts));
			} else {
				modal.resolve(await apiCreateServer(opts));
			}
			await modal.hide();
		},
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate(schema.parse(value));
		},
		defaultValues,
		validators: {
			onSubmit: schema,
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
					{server?.server_id ? "Edit" : "Create"} Server
				</DialogTitle>

				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 4 }}>
							<form.AppField
								name={"short_name"}
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
								name={"address_internal"}
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
								name={"sdr_enabled"}
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
								name={"log_secret"}
								children={(field) => {
									return <field.TextField label={"Log Secret"} />;
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
									return <field.TextField label={"Latitude"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"longitude"}
								children={(field) => {
									return <field.TextField label={"Longitude"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 4 }}>
							<form.AppField
								name={"reserved_slots"}
								children={(field) => {
									return <field.TextField label={"Reserved Slots"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 4 }}>
							<form.AppField
								name={"is_enabled"}
								children={(field) => {
									return <field.CheckboxField label={"Is Enabled"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 4 }}>
							<form.AppField
								name={"enabled_stats"}
								children={(field) => {
									return <field.CheckboxField label={"Stats Enabled"} />;
								}}
							/>
						</Grid>

						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"discord_seed_role_ids"}
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
