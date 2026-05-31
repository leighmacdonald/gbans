import { create } from "@bufbuild/protobuf";
import { durationFromMs } from "@bufbuild/protobuf/wkt";
import { useMutation } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import DirectionsRunIcon from "@mui/icons-material/DirectionsRun";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { differenceInSeconds } from "date-fns/fp/differenceInSeconds";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { BanReason, BanType, CreateRequestSchema, Origin } from "../../rpc/ban/v1/ban_pb.ts";
import { create as createBan } from "../../rpc/ban/v1/ban-BanService_connectquery.ts";
import { enumValues } from "../../util/lists.ts";
import { banTypeString } from "../../util/strings.ts";
import { zeroStringUndefined } from "../../util/types.ts";
import { MarkdownField } from "../form/field/MarkdownField.tsx";
import { Heading } from "../Heading.tsx";

export const BanCreateModal = NiceModal.create(({ reportId, steamId }: { reportId?: number; steamId?: string }) => {
	const { sendFlash, sendError } = useUserFlashCtx();
	const modal = useModal();

	const mutation = useMutation(createBan, {
		onSuccess: async (banRecord) => {
			sendFlash("success", "Created ban successfully");
			modal.resolve(banRecord);
			await modal.hide();
		},
		onError: sendError,
	});

	const defaultValues = {
		reportId: reportId ?? 0,
		targetId: steamId ?? "",
		banType: BanType.BANNED,
		reason: BanReason.CHEATING,
		reasonText: "",
		note: "",
		evadeOk: false,
		cidr: "",
		demoName: "",
		demoTick: 0,
		origin: Origin.REPORTED,
		validUntil: new Date(),
	};

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate(
				create(CreateRequestSchema, {
					banType: value.banType,
					reason: value.reason,
					reasonText: zeroStringUndefined(value.note),
					note: zeroStringUndefined(value.note),
					evadeOk: value.evadeOk,
					duration: durationFromMs(differenceInSeconds(value.validUntil, new Date()) * 1000),
					cidr: zeroStringUndefined(value.cidr),
					origin: Origin.WEB,
					targetId: value.targetId,
					demoTick: value.demoTick,
					demoName: zeroStringUndefined(value.demoName),
					reportId: reportId,
				}),
			);
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
					Create Ban
				</DialogTitle>

				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"targetId"}
								children={(field) => {
									return <field.SteamIDField label={"Target Steam ID Or Group ID"} />;
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
								name={"banType"}
								children={(field) => {
									return (
										<field.BanTypeField
											label={"Ban Action Type"}
											items={enumValues(BanType)}
											renderItem={(bt) => {
												return (
													<MenuItem value={bt} key={`bt-${bt}`}>
														{banTypeString(bt)}
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
										<field.BanReasonField
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
								name={"reasonText"}
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
						<Grid>
							<form.AppField
								name={"validUntil"}
								children={(field) => {
									return <field.DateTimeField label={"Expires At"} />;
								}}
							/>
						</Grid>

						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"evadeOk"}
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
});
