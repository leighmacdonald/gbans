import { useMutation } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import GroupsIcon from "@mui/icons-material/Groups";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { type Group, type GroupOverrides, OverrideAccess, OverrideType } from "../../rpc/sourcemod/v1/sourcemod_pb.ts";
import {
	createGroupOverride,
	editGroupOverride,
} from "../../rpc/sourcemod/v1/sourcemod-SourcemodService_connectquery.ts";
import { Heading } from "../Heading";

export const SMGroupOverrideEditorModal = NiceModal.create(
	({ group, override }: { group: Group; override?: GroupOverrides }) => {
		const modal = useModal();
		const { sendError } = useUserFlashCtx();
		const createMutation = useMutation(createGroupOverride, {
			onSuccess: async (override) => {
				modal.resolve(override);
				await modal.hide();
			},
			onError: sendError,
		});

		const editMutation = useMutation(editGroupOverride, {
			onSuccess: async (override) => {
				modal.resolve(override);
				await modal.hide();
			},
			onError: sendError,
		});

		const form = useAppForm({
			onSubmit: async ({ value }) => {
				if (override?.groupOverrideId) {
					editMutation.mutate({ ...value, groupOverrideId: override.groupOverrideId });
				} else {
					createMutation.mutate({ ...value, groupId: group.groupId });
				}
			},
			defaultValues: {
				type: override?.overrideType ?? OverrideType.COMMAND_UNSPECIFIED,
				name: override?.name ?? "",
				access: override?.overrideAccess ?? OverrideAccess.ALLOW_UNSPECIFIED,
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
						{override ? "Edit" : "Create"} Group Override
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
											<field.SelectField
												label={"Override Type"}
												items={["command", "group"]}
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

							<Grid size={{ xs: 6 }}>
								<form.AppField
									name={"access"}
									children={(field) => {
										return (
											<field.SelectField
												label={"Access Type"}
												items={["allow", "deny"]}
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
	},
);
