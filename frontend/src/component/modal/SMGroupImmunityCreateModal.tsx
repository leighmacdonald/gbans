import { useMutation } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import GroupsIcon from "@mui/icons-material/Groups";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import type { Group } from "../../rpc/sourcemod/v1/sourcemod_pb.ts";
import { createImmunity } from "../../rpc/sourcemod/v1/sourcemod-SourcemodService_connectquery.ts";
import { Heading } from "../Heading";

export const SMGroupImmunityCreateModal = NiceModal.create(({ groups }: { groups: Group[] }) => {
	const modal = useModal();
	const { sendError } = useUserFlashCtx();

	const mutation = useMutation(createImmunity, {
		onSuccess: async (immunity) => {
			modal.resolve(immunity.groupImmunity);
			await modal.hide();
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate({ groupId: value.group.groupId, otherId: value.other.groupId });
		},
		defaultValues: {
			group: groups[0],
			other: groups[1],
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
					Select Group
				</DialogTitle>

				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"group"}
								children={(field) => {
									return (
										<field.SelectField
											label={"Group"}
											items={groups}
											renderItem={(i) => {
												if (!i) {
													return;
												}
												return (
													<MenuItem value={i.groupId} key={i.groupId}>
														{i.name}
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
								name={"other"}
								children={(field) => {
									return (
										<field.SelectField
											label={"Immunity From"}
											items={groups}
											renderItem={(i) => {
												if (!i) {
													return;
												}
												return (
													<MenuItem value={i.groupId} key={i.groupId}>
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
