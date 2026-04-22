import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import EmojiEventsIcon from "@mui/icons-material/EmojiEvents";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { Heading } from "../Heading";
import { Privilege } from "../../rpc/person/v1/privilege_pb.ts";
import { useMutation } from "@connectrpc/connect-query";
import { contestCreate } from "../../rpc/contest/v1/contest-Service_connectquery.ts";
import type { Contest } from "../../rpc/contest/v1/contest_pb.ts";
import { enumValues } from "../../util/lists.ts";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { EMPTY_UUID } from "../../util/strings.ts";

export const ContestEditor = NiceModal.create(({ contest }: { contest?: Contest }) => {
	const modal = useModal();
	const { sendError } = useUserFlashCtx();

	const mutation = useMutation(contestCreate, {
		onSuccess: async (contest) => {
			modal.resolve(contest);
			await modal.hide();
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate({ contest: value });
		},
		defaultValues: {
			date_start: contest?.dateStart ? timestampDate(contest?.dateStart).toISOString() : "",
			date_end: contest?.dateEnd ? timestampDate(contest.dateEnd).toISOString() : "",
			description: contest ? contest.description : "",
			hide_submissions: contest ? contest.hideSubmissions : false,
			title: contest ? contest.title : "",
			voting: contest ? contest.voting : true,
			down_votes: contest ? contest.downVotes : true,
			max_submissions: contest ? String(contest.maxSubmissions) : "1",
			media_types: contest ? contest.mediaTypes : "",
			public: contest ? contest.public : true,
			min_permission_level: contest ? contest.minPermissionLevel : Privilege.USER,
			//deleted: contest ? contest.deleted : false,
			num_entries: 0,
			updated_on: new Date(),
			created_on: new Date(),
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
				<DialogTitle component={Heading} iconLeft={<EmojiEventsIcon />}>
					{`${contest?.contestId === EMPTY_UUID ? "Create" : "Edit"} A Contest`}
				</DialogTitle>

				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"title"}
								validators={{
									onChange: z.string().min(5),
								}}
								children={(field) => {
									return <field.TextField label={"Title"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"description"}
								validators={{
									onChange: z.string().min(5),
								}}
								children={(field) => {
									return <field.MarkdownField label={"Description"} rows={10} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 4 }}>
							<form.AppField
								name={"public"}
								validators={{
									onChange: z.boolean(),
								}}
								children={(field) => {
									return <field.CheckboxField label={"Public"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 4 }}>
							<form.AppField
								name={"hide_submissions"}
								validators={{
									onChange: z.boolean(),
								}}
								children={(field) => {
									return <field.CheckboxField label={"Hide Submissions"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 4 }}>
							<form.AppField
								name={"max_submissions"}
								children={(field) => {
									return <field.TextField label={"Max Submissions"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"min_permission_level"}
								children={(field) => {
									return (
										<field.SelectField
											label={"Min Permissions"}
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
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"voting"}
								validators={{
									onChange: z.boolean(),
								}}
								children={(field) => {
									return <field.CheckboxField label={"Voting Enabled"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"down_votes"}
								validators={{
									onChange: z.boolean(),
								}}
								children={(field) => {
									return <field.CheckboxField label={"Downvotes Enabled"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"date_start"}
								children={(field) => {
									return <field.DateTimeField label={"Custom Expire Date"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"date_end"}
								children={(field) => {
									return <field.DateTimeField label={"Custom Expire Date"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 6 }}>
							<form.AppField
								name={"media_types"}
								validators={{
									onChange: z.string().refine((arg) => {
										if (arg === "") {
											return true;
										}

										const parts = arg?.split(",");
										const matches = parts.filter((p) => p.match(/^\S+\/\S+$/));
										return matches.length === parts.length;
									}),
								}}
								children={(field) => {
									return <field.TextField label={"Allowed Mime Types"} />;
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
