import { create } from "@bufbuild/protobuf";
import { DurationSchema, timestampDate } from "@bufbuild/protobuf/wkt";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import DirectionsRunIcon from "@mui/icons-material/DirectionsRun";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { formatDuration, formatISO9075 } from "date-fns";
import { intervalToDuration } from "date-fns/intervalToDuration";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { BanReason, BanType, Origin } from "../../rpc/ban/v1/ban_pb.ts";
import { get, update } from "../../rpc/ban/v1/ban-BanService_connectquery.ts";
import { enumValues } from "../../util/lists.ts";
import { ErrorDetails } from "../ErrorDetails.tsx";
import { MarkdownField } from "../form/field/MarkdownField.tsx";
import { Heading } from "../Heading";
import { LoadingPlaceholder } from "../LoadingPlaceholder.tsx";

export const BanModal = NiceModal.create(
	({ banId, reportId, steamId }: { banId?: number; reportId?: number; steamId?: bigint }) => {
		const { data: req, isLoading, isError, error } = useQuery(get, { banId });

		const { sendFlash, sendError } = useUserFlashCtx();
		const modal = useModal();

		const mutation = useMutation(update, {
			onSuccess: async (banRecord) => {
				if (req?.ban?.banId) {
					sendFlash("success", "Updated ban successfully");
				} else {
					sendFlash("success", "Created ban successfully");
				}
				modal.resolve(banRecord);
				await modal.hide();
			},
			onError: sendError,
		});

		const defaultValues = {
			report_id: req?.ban?.reportId ?? reportId ?? 0,
			target_id: req?.ban?.targetId ?? steamId ?? "",
			ban_type: req?.ban?.banType ?? BanType.BANNED,
			reason: req?.ban?.reason ?? BanReason.CHEATING,
			reason_text: req?.ban?.reasonText ?? "",
			note: req?.ban?.note ?? "",
			evade_ok: req?.ban?.evadeOk ?? false,
			cidr: req?.ban?.cidr ?? "",
			demo_name: "",
			demo_tick: 0,
			origin: req?.ban?.origin ?? Origin.REPORTED,
		};

		const form = useAppForm({
			onSubmit: async ({ value }) => {
				mutation.mutate({ ...value, duration: create(DurationSchema, { seconds: 100n }) });
				throw "fixme";
				//const seconds = BigInt(toSeconds(parse(value.duration)));
			},
			defaultValues,
		});

		if (isLoading) {
			return <LoadingPlaceholder />;
		}

		if (isError) {
			return <ErrorDetails error={error} />;
		}

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
						{Number(banId) > 0 ? "Edit Ban" : "Create Ban"}
					</DialogTitle>

					<DialogContent>
						<Grid container spacing={2}>
							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"target_id"}
									children={(field) => {
										return (
											<field.SteamIDField
												label={"Target Steam ID Or Group ID"}
												disabled={Boolean(req?.ban?.banId)}
											/>
										);
									}}
								/>
							</Grid>

							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"cidr"}
									children={(field) => {
										return <field.TextField label={"IP/CIDR"} />;
									}}
								/>
							</Grid>

							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"ban_type"}
									children={(field) => {
										return (
											<field.SelectField
												label={"Ban Action Type"}
												items={enumValues(BanType)}
												renderItem={(bt) => {
													return (
														<MenuItem value={bt} key={`bt-${bt}`}>
															{BanType[bt]}
														</MenuItem>
													);
												}}
											/>
										);
									}}
								/>
							</Grid>

							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"reason"}
									children={(field) => {
										return (
											<field.SelectField
												label={"Reason"}
												items={enumValues(BanReason)}
												renderItem={(br) => {
													return (
														<MenuItem value={br} key={`br-${br}`}>
															{BanReason[br]}
														</MenuItem>
													);
												}}
											/>
										);
									}}
								/>
							</Grid>
							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"reason_text"}
									validators={{
										onSubmit: ({ value, fieldApi }) => {
											if (fieldApi.form.getFieldValue("reason") !== BanReason.CUSTOM) {
												if (value.length === 0) {
													return undefined;
												}
												return "Must use custom ban reason";
											}
											const result = z.string().min(5).safeParse(value);
											if (!result.success) {
												return result.error.message;
											}

											return undefined;
										},
									}}
									children={(field) => {
										return <field.TextField label={"Custom Ban Reason"} />;
									}}
								/>
							</Grid>

							<Grid size={{ xs: 12 }}>
								{req?.ban?.validUntil && (
									<>
										<p>
											Expires In:{" "}
											{formatDuration(
												intervalToDuration({
													start: new Date(),
													end: timestampDate(req?.ban.validUntil),
												}),
											)}
										</p>
										<p>Expires On: {formatISO9075(timestampDate(req?.ban.validUntil))}</p>
									</>
								)}
								{/*<form.AppField*/}
								{/*	name={"duration"}*/}
								{/*	children={(field) => {*/}
								{/*		return (*/}
								{/*			<field.SelectField*/}
								{/*				label={"Duration"}*/}
								{/*				items={Object.values(Duration)}*/}
								{/*				renderItem={(bt) => {*/}
								{/*					return (*/}
								{/*						<MenuItem value={bt} key={`bt-${bt}`}>*/}
								{/*							{Duration8601ToString(bt)}*/}
								{/*						</MenuItem>*/}
								{/*					);*/}
								{/*				}}*/}
								{/*			/>*/}
								{/*		);*/}
								{/*	}}*/}
								{/*/>*/}
							</Grid>

							{/*<Grid size={{ xs: 6 }}>*/}
							{/*    <form.AppField*/}
							{/*        name={'duration'}*/}
							{/*        children={(field) => {*/}
							{/*            return <field.TextField label={'Duration'} />;*/}
							{/*        }}*/}
							{/*    />*/}
							{/*</Grid>*/}

							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"evade_ok"}
									children={(field) => {
										return <field.CheckboxField label={"IP Evading Allowed"} />;
									}}
								/>
							</Grid>

							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"note"}
									children={(props) => {
										return (
											<MarkdownField
												{...props}
												value={props.state.value}
												multiline={true}
												rows={10}
												label={"Mod Notes"}
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
	},
);
