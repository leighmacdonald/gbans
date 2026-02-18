import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import GroupsIcon from "@mui/icons-material/Groups";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { useMutation } from "@tanstack/react-query";
import "video-react/dist/video-react.css";
import { apiCreateSMGroupOverrides, apiSaveSMGroupOverrides } from "../../api";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import type { OverrideAccess, OverrideType, SMGroupOverrides, SMGroups } from "../../schema/sourcemod.ts";
import { Heading } from "../Heading";

type mutateOverrideArgs = {
	name: string;
	type: OverrideType;
	access: OverrideAccess;
};

export const SMGroupOverrideEditorModal = NiceModal.create(
	({ group, override }: { group: SMGroups; override?: SMGroupOverrides }) => {
		const modal = useModal();
		const { sendError } = useUserFlashCtx();
		const mutation = useMutation({
			mutationKey: ["adminSMGroupOverride"],
			mutationFn: async ({ name, type, access }: mutateOverrideArgs) => {
				return override?.group_override_id
					? await apiSaveSMGroupOverrides(override.group_override_id, name, type, access)
					: await apiCreateSMGroupOverrides(group.group_id, name, type, access);
			},
			onSuccess: async (override) => {
				modal.resolve(override);
				await modal.hide();
			},
			onError: sendError,
		});

		const form = useAppForm({
			onSubmit: async ({ value }) => {
				mutation.mutate(value);
			},
			defaultValues: {
				type: override?.type ?? "command",
				name: override?.name ?? "",
				access: override?.access ?? "allow",
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
