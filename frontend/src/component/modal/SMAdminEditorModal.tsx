import { useMutation } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import GroupsIcon from "@mui/icons-material/Groups";
import ButtonGroup from "@mui/material/ButtonGroup";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import MenuItem from "@mui/material/MenuItem";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { Permission } from "../../rpc/roles/v1/roles_pb.ts";
import { type Admin, AuthType } from "../../rpc/sourcemod/v1/sourcemod_pb.ts";
import { createAdmin, editAdmin } from "../../rpc/sourcemod/v1/sourcemod-SourcemodService_connectquery.ts";
import { enumValues } from "../../util/lists.ts";
import { smPermissions } from "../../util/strings.ts";
import { Heading } from "../Heading";

const schema = z.object({
	name: z.string().min(2),
	password: z.string(),
	authType: z.enum(AuthType),
	identity: z.string().min(1),
	permissions: z.array(z.number()),
});

export const SMAdminEditorModal = NiceModal.create(({ admin }: { admin?: Admin }) => {
	const modal = useModal();
	const { sendError, sendFlash } = useUserFlashCtx();
	const defaultValues: z.input<typeof schema> = {
		authType:
			admin?.authType === "steam"
				? AuthType.STEAM_UNSPECIFIED
				: admin?.authType === "name"
					? AuthType.NAME
					: admin?.authType === "ip"
						? AuthType.IP
						: AuthType.STEAM_UNSPECIFIED,
		identity: admin?.identity ?? "",
		password: admin?.password ?? "",
		name: admin?.name ?? "",
		permissions: admin?.permissions ?? [],
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
			if (admin?.adminId) {
				editMutation.mutate({
					adminId: admin.adminId,
					...value,
				});
			} else {
				createMutation.mutate(value);
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
										<field.SelectAuthTypeField
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
								name={"permissions"}
								children={(field) => {
									return (
										<field.SelectPermissionsField
											multiple={true}
											label={"Permissions"}
											items={smPermissions}
											renderItem={(p) => {
												return (
													<MenuItem value={p} key={`perm-${p}`}>
														{Permission[p]}
													</MenuItem>
												);
											}}
										/>
									);
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
