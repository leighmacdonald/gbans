import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import FilterAltIcon from "@mui/icons-material/FilterAlt";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { Heading } from "../Heading";
import { type Filter, FilterAction } from "../../rpc/chat/v1/wordfilter_pb.ts";
import { useMutation } from "@connectrpc/connect-query";
import { filterCreate } from "../../rpc/chat/v1/wordfilter-WordfilterService_connectquery.ts";
import { enumValues } from "../../util/lists.ts";

const schema = z.object({
	pattern: z.string({ message: "Must entry pattern" }).min(2),
	is_regex: z.boolean(),
	action: z.enum(FilterAction),
	duration: z.string({ message: "Must provide a duration" }),
	weight: z.number().min(1).max(100),
	is_enabled: z.boolean(),
});

export const FilterEditModal = NiceModal.create(({ filter }: { filter?: Filter }) => {
	const modal = useModal();
	const { sendError } = useUserFlashCtx();
	const defaultValues: z.input<typeof schema> = {
		pattern: filter ? String(filter.pattern) : "",
		is_regex: filter?.isRegex ?? false,
		is_enabled: filter?.isEnabled ?? true,
		action: filter?.action ?? FilterAction.KICK_UNSPECIFIED,
		duration: filter?.duration ?? "1w",
		weight: filter ? filter.weight : 1,
	};
	const mutation = useMutation(filterCreate, {
		onSuccess: async (result) => {
			modal.resolve(result);
			await modal.hide();
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async (value) => {
			// FIXME add edit mutation
			mutation.mutate({ filter: value.value });
		},
		defaultValues,
		validators: {
			onSubmit: schema,
		},
	});

	return (
		<Dialog {...muiDialogV5(modal)} fullWidth maxWidth={"md"}>
			<form
				onSubmit={async (e) => {
					e.preventDefault();
					e.stopPropagation();
					await form.handleSubmit();
				}}
			>
				<DialogTitle component={Heading} iconLeft={<FilterAltIcon />}>
					Filter Editor
				</DialogTitle>
				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 8 }}>
							<form.AppField
								name={"pattern"}
								children={(field) => {
									return <field.TextField label={"Pattern"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 4 }}>
							<form.AppField
								name={"is_regex"}
								children={(field) => {
									return <field.CheckboxField label={"Is Regex Pattern"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 4 }}>
							<form.AppField
								name={"action"}
								children={(field) => {
									return (
										<field.SelectField
											label={"Action"}
											items={enumValues(FilterAction)}
											renderItem={(fa) => {
												return (
													<MenuItem value={fa} key={`fa-${fa}`}>
														{FilterAction[fa]}
													</MenuItem>
												);
											}}
										/>
									);
								}}
							/>
						</Grid>
						<Grid size={{ xs: 4 }}>
							<form.AppField
								name={"duration"}
								children={(field) => {
									return <field.TextField label={"Duration"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 4 }}>
							<form.AppField
								name={"weight"}
								children={(field) => {
									return <field.NumberField label={"Weight"} min={1} max={100} />;
								}}
							/>
						</Grid>

						<Grid size={{ xs: 4 }}>
							<form.AppField
								name={"is_enabled"}
								validators={{
									onSubmit: z.boolean(),
								}}
								children={(field) => {
									return <field.CheckboxField label={"Is Enabled"} />;
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
