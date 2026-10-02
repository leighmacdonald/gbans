import { createConnectQueryKey, useMutation, useQuery, useTransport } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import PersonAddAltIcon from "@mui/icons-material/PersonAddAlt";
import ButtonGroup from "@mui/material/ButtonGroup";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { useQueryClient } from "@tanstack/react-query";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { type Role, RolesService } from "../../rpc/roles/v1/roles_pb.ts";
import { roleList, setUserRoles } from "../../rpc/roles/v1/roles-RolesService_connectquery.ts";
import { Heading } from "../Heading.tsx";

type AssignRolesFormValues = {
	steamId: string;
	roleIds: number[];
};

export const AssignRolesModal = NiceModal.create(({ steamId, roles }: { steamId?: string; roles?: Role[] }) => {
	const { sendFlash, sendError } = useUserFlashCtx();
	const modal = useModal();
	const queryClient = useQueryClient();
	const transport = useTransport();

	const isEdit = Boolean(steamId);

	const { data: rolesResponse } = useQuery(roleList, {});
	const allRoles = rolesResponse?.roles ?? [];

	const resetQueries = () => {
		queryClient.invalidateQueries({
			queryKey: createConnectQueryKey({
				schema: RolesService.method.roleUsers,
				cardinality: "finite",
				transport,
				input: {},
			}),
		});
		queryClient.invalidateQueries({
			queryKey: createConnectQueryKey({
				schema: RolesService.method.roleList,
				cardinality: "finite",
				transport,
				input: {},
			}),
		});
	};

	const mutation = useMutation(setUserRoles, {
		onSuccess: () => {
			sendFlash("success", "User roles updated successfully");
			resetQueries();
			modal.resolve(true);
			void modal.hide();
		},
		onError: sendError,
	});

	const defaultValues: AssignRolesFormValues = {
		steamId: steamId ?? "",
		roleIds: (roles ?? []).map((r) => r.roleId),
	};

	const form = useAppForm<AssignRolesFormValues>({
		onSubmit: async ({ value }) => {
			mutation.mutate({
				steamId: value.steamId,
				roleId: value.roleIds,
			});
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
				<DialogTitle component={Heading} iconLeft={<PersonAddAltIcon />}>
					{isEdit ? `Edit Roles: ${steamId}` : "Assign Roles"}
				</DialogTitle>

				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"steamId"}
								children={(field) => {
									return (
										<field.SteamIDField
											label={"Steam ID"}
											disabled={isEdit}
											defaultSteamID={steamId}
										/>
									);
								}}
							/>
						</Grid>

						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"roleIds"}
								children={(field) => {
									return (
										<field.SelectField
											multiple={true}
											label={"Assigned Roles"}
											items={allRoles.map((r) => r.roleId)}
											renderItem={(roleId) => {
												const role = allRoles.find((r) => r.roleId === roleId);
												return (
													<MenuItem value={roleId} key={`role-${roleId}`}>
														{role?.roleName ?? roleId}
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
									<form.CloseButton onClick={() => modal.hide()} />
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
