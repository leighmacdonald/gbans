import { useMutation } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import GroupsIcon from "@mui/icons-material/Groups";
import ButtonGroup from "@mui/material/ButtonGroup";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { Permission } from "../../rpc/roles/v1/roles_pb.ts";
import { type Override, OverrideType } from "../../rpc/sourcemod/v1/sourcemod_pb.ts";
import { createOverrides, editOverrides } from "../../rpc/sourcemod/v1/sourcemod-SourcemodService_connectquery.ts";
import { enumValues } from "../../util/lists.ts";
import { smPermissions } from "../../util/strings.ts";
import { Heading } from "../Heading";

const schema = z.object({
	name: z.string(),
	type: z.enum(OverrideType),
	permissions: z.array(z.number()),
});

export const SMOverrideEditorModal = NiceModal.create(({ override }: { override?: Override }) => {
	const modal = useModal();
	const { sendError } = useUserFlashCtx();

	const defaultValues: z.input<typeof schema> = {
		type: override?.overrideType ?? OverrideType.COMMAND_UNSPECIFIED,
		name: override?.name ?? "",
		permissions: override?.permissions ?? [],
	};

	const createMutation = useMutation(createOverrides, {
		onSuccess: async (admin) => {
			modal.resolve(admin);
			await modal.hide();
		},
		onError: sendError,
	});

	const editMutation = useMutation(editOverrides, {
		onSuccess: async (admin) => {
			modal.resolve(admin);
			await modal.hide();
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			if (Number(override?.overrideId) > 0) {
				editMutation.mutate({ overrideId: override.overrideId, ...value });
			} else {
				createMutation.mutate(value);
			}
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
				<DialogTitle component={Heading} iconLeft={<GroupsIcon />}>
					{override ? "Edit" : "Create"} Override
				</DialogTitle>

				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"name"}
								children={(field) => {
									return <field.TextField label={"Name"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"type"}
								children={(field) => {
									return (
										<field.SelectOverrideTypeField
											label={"Override Type"}
											items={enumValues(OverrideType)}
											renderItem={(i) => {
												return (
													<MenuItem value={i} key={i}>
														{i}
													</MenuItem>
												);
											}}
										/>
									);
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
