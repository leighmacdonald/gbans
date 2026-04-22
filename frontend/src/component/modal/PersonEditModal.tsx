import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import PersonIcon from "@mui/icons-material/Person";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { useAppForm } from "../../contexts/formContext.tsx";

import { Heading } from "../Heading";
import type { Person } from "../../rpc/person/v1/person_pb.ts";
import { useMutation } from "@connectrpc/connect-query";
import { editPermissions } from "../../rpc/person/v1/person-PersonService_connectquery.ts";
import { Privilege } from "../../rpc/person/v1/privilege_pb.ts";
import { enumValues } from "../../util/lists.ts";

export const PersonEditModal = NiceModal.create(({ person }: { person: Person }) => {
	const modal = useModal();

	const mutation = useMutation(editPermissions, {
		onSuccess: async (response) => {
			modal.resolve(response.person);
			await modal.hide();
		},
		onError: async (err) => {
			modal.reject(err);
			await modal.hide();
		},
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate(value);
		},
		defaultValues: {
			permissionLevel: person.permissionLevel,
		},
	});

	return (
		<Dialog {...muiDialogV5(modal)} fullWidth maxWidth={"sm"}>
			<form
				onSubmit={async (e) => {
					e.preventDefault();
					e.stopPropagation();
					await form.handleSubmit();
				}}
			>
				<DialogTitle component={Heading} iconLeft={<PersonIcon />}>
					Person Editor: {person.personaName}
				</DialogTitle>
				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"permissionLevel"}
								children={(field) => {
									return (
										<field.SelectField
											label={"Permissions"}
											items={enumValues(Privilege)}
											renderItem={(pl) => {
												return (
													<MenuItem value={pl} key={`pl-${pl}`}>
														{Privilege[pl]}
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
