import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import GroupsIcon from "@mui/icons-material/Groups";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import "video-react/dist/video-react.css";
import { useAppForm } from "../../contexts/formContext.tsx";
import type { SMGroups } from "../../schema/sourcemod.ts";
import { Heading } from "../Heading";

export const SMGroupSelectModal = NiceModal.create(({ groups }: { groups: SMGroups[] }) => {
	const modal = useModal();

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			// TODO fix typing for select field and objects
			const group = groups.find((v) => v.group_id === (value.group as unknown as number));
			if (group) {
				modal.resolve(group);
			} else {
				modal.reject("Invalid group selected");
			}
			await modal.hide();
		},
		defaultValues: {
			group: groups[0],
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
						<Grid size={{ xs: 12 }}>
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
													<MenuItem value={i.group_id} key={i.group_id}>
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
