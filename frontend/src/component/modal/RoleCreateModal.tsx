import { create } from "@bufbuild/protobuf";
import { useMutation } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import DirectionsRunIcon from "@mui/icons-material/DirectionsRun";
import ButtonGroup from "@mui/material/ButtonGroup";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { Permission, type RoleCreateRequest, RoleCreateRequestSchema } from "../../rpc/roles/v1/roles_pb.ts";
import { roleCreate } from "../../rpc/roles/v1/roles-RolesService_connectquery.ts";
import { enumValues } from "../../util/lists.ts";
import { Heading } from "../Heading.tsx";

export const RoleCreateModal = NiceModal.create(() => {
	const { sendError } = useUserFlashCtx();
	const modal = useModal();

	const mutation = useMutation(roleCreate, {
		onSuccess: async (role) => {
			modal.resolve(role);
			await modal.hide();
		},
		onError: sendError,
	});

	const defaultValues: Omit<RoleCreateRequest, "$typeName"> = {
		roleName: "",
		permissions: [],
	};

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate(create(RoleCreateRequestSchema, value));
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
				<DialogTitle component={Heading} iconLeft={<DirectionsRunIcon />}>
					Create Role
				</DialogTitle>

				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"roleName"}
								children={(field) => {
									return <field.TextField label={"Role Name"} />;
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
											label={"Assigned Permissions"}
											items={enumValues(Permission)}
											renderItem={(bt) => {
												return (
													<MenuItem value={bt} key={`bt-${bt}`}>
														{Permission[bt]}
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
