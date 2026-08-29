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
import type { Group } from "../../rpc/sourcemod/v1/sourcemod_pb.ts";
import { createGroup, editGroups } from "../../rpc/sourcemod/v1/sourcemod-SourcemodService_connectquery.ts";
import { smPermissions } from "../../util/strings.ts";
import { Heading } from "../Heading";

const schema = z.object({
	name: z.string().min(2),
	permissions: z.array(z.number()),
});

export const SMGroupEditorModal = NiceModal.create(({ group }: { group?: Group }) => {
	const modal = useModal();
	const { sendError } = useUserFlashCtx();
	const defaultValues: z.input<typeof schema> = {
		name: group?.name ?? "",
		permissions: group?.permissions ?? [],
	};
	const createMutation = useMutation(createGroup, {
		onSuccess: async (group) => {
			modal.resolve(group);
			await modal.hide();
		},
		onError: sendError,
	});

	const editMutation = useMutation(editGroups, {
		onSuccess: async (group) => {
			modal.resolve(group);
			await modal.hide();
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			if (group?.groupId) {
				editMutation.mutate({ groupId: group.groupId, ...value });
			} else {
				createMutation.mutate(value);
			}
		},
		validators: {
			onSubmit: schema,
		},
		defaultValues,
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
					SM Group Editor
				</DialogTitle>

				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"name"}
								children={(field) => {
									return <field.TextField label={"Group Name"} />;
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
