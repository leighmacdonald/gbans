import { useMutation } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import GroupsIcon from "@mui/icons-material/Groups";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import MenuItem from "@mui/material/MenuItem";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { AuthType, type Group, type SMUser } from "../../rpc/sourcemod/v1/sourcemod_pb.ts";
import { createAdmin, editAdmin } from "../../rpc/sourcemod/v1/sourcemod-SourcemodService_connectquery.ts";
import { enumValues } from "../../util/lists.ts";
import { hasSMFlag, schemaFlags } from "../../util/strings.ts";
import { Heading } from "../Heading";

const schema = schemaFlags.extend({
	name: z.string().min(2),
	password: z.string(),
	authType: z.enum(AuthType),
	identity: z.string().min(1),
	immunity: z.number().min(0).max(100),
});

export const SMAdminEditorModal = NiceModal.create(({ admin }: { admin?: SMUser; groups: Group[] }) => {
	const modal = useModal();
	const { sendError, sendFlash } = useUserFlashCtx();
	const defaultValues: z.input<typeof schema> = {
		authType: admin?.authType ?? AuthType.STEAM_UNSPECIFIED,
		identity: admin?.identity ?? "",
		password: admin?.password ?? "",
		name: admin?.name ?? "",
		immunity: admin?.immunity ?? 0,
		z: hasSMFlag("z", admin),
		a: hasSMFlag("a", admin),
		b: hasSMFlag("b", admin),
		c: hasSMFlag("c", admin),
		d: hasSMFlag("d", admin),
		e: hasSMFlag("e", admin),
		f: hasSMFlag("f", admin),
		g: hasSMFlag("g", admin),
		h: hasSMFlag("h", admin),
		i: hasSMFlag("i", admin),
		j: hasSMFlag("j", admin),
		k: hasSMFlag("k", admin),
		l: hasSMFlag("l", admin),
		m: hasSMFlag("m", admin),
		n: hasSMFlag("n", admin),
		o: hasSMFlag("o", admin),
		p: hasSMFlag("p", admin),
		q: hasSMFlag("q", admin),
		r: hasSMFlag("r", admin),
		s: hasSMFlag("s", admin),
		t: hasSMFlag("t", admin),
	};

	const createMutation = useMutation(createAdmin, {
		onSuccess: async (resp) => {
			modal.resolve(resp.admin);
			await modal.hide();
			sendFlash("success", "Created admin");
		},
		onError: (error) => {
			sendError(error.message);
		},
	});

	const editMutation = useMutation(editAdmin, {
		onSuccess: async (resp) => {
			modal.resolve(resp.admin);
			await modal.hide();
			sendFlash("success", "Edited admin");
		},
		onError: (error) => {
			sendError(error.message);
		},
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			const filteredKeys = ["name", "immunity", "auth_type", "password", "identity"];
			const flags = Object.entries(value)
				.filter((v) => !filteredKeys.includes(v[0]))
				.reduce((acc, value) => {
					if (value[1]) {
						acc += value[0];
					}
					return acc;
				}, "");
			if (admin?.id) {
				editMutation.mutate({
					...value,
					immunity: Number(value.immunity),
					flags: flags,
				});
			} else {
				createMutation.mutate({ ...value });
			}
		},
		defaultValues,
		validators: {
			onChange: schema,
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
				<DialogTitle component={Heading} iconLeft={<GroupsIcon />}>
					SM Admin Editor
				</DialogTitle>

				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"name"}
								children={(field) => {
									return <field.TextField label={"Alias"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"password"}
								validators={{}}
								children={(field) => {
									return <field.TextField label={"Password"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"authType"}
								children={(field) => {
									return (
										<field.SelectField
											label={"Auth Type"}
											items={enumValues(AuthType)}
											renderItem={(i) => {
												return (
													<MenuItem value={i} key={i}>
														{AuthType[i]}
													</MenuItem>
												);
											}}
										/>
									);
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"identity"}
								children={(field) => {
									return <field.TextField label={"Identity"} />;
								}}
							/>
						</Grid>

						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"immunity"}
								children={(field) => {
									return <field.TextField label={"Immunity Level"} />;
								}}
							/>
						</Grid>

						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"z"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(z) Full Admin"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"a"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(a) Reserved Slot"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"b"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(b) Generic Admin"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"c"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(c) Kick Players"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"d"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(d) Ban Players"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"e"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(e) Unban Players"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"f"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(f) Slay/Harm Players"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"g"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(g) Change Maps"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"h"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(h) Change CVARs"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"i"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(i) Exec Configs"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"j"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(j) Special Chat Privileges"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"k"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(k) Start Votes"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"l"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(l) Set Server Password"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"m"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(m) RCON Access"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"n"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(n) Enabled Cheats"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"o"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(o) Custom Flag"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"p"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(p) Custom Flag"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"q"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(q) Custom Flag"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"r"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(r) Custom Flag"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"s"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(s) Custom Flag"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"t"}
								validators={{ onChange: z.boolean() }}
								children={(field) => {
									return <field.CheckboxField label={"(t) Custom Flag"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 12 }}>
							<Link target={"_blank"} href={"https://wiki.alliedmods.net/Adding_Admins_(SourceMod)"}>
								Additional SourceMod Admin Info
							</Link>
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
