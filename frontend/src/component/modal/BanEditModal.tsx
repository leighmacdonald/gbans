import { create } from "@bufbuild/protobuf";
import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import DirectionsRunIcon from "@mui/icons-material/DirectionsRun";
import { Dialog, DialogActions, DialogContent, DialogTitle, Typography } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { formatDuration, formatISO9075, isAfter } from "date-fns";
import { intervalToDuration } from "date-fns/intervalToDuration";
import { useMemo } from "react";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { AppealState, BanReason, BanType, type UpdateRequest, UpdateRequestSchema } from "../../rpc/ban/v1/ban_pb.ts";
import { get, update } from "../../rpc/ban/v1/ban-BanService_connectquery.ts";
import { enumValues } from "../../util/lists.ts";
import { banTypeString } from "../../util/strings.ts";
import { emptyOrNullString } from "../../util/types.ts";
import { ErrorDetails } from "../ErrorDetails.tsx";
import { MarkdownField } from "../form/field/MarkdownField.tsx";
import { Heading } from "../Heading.tsx";
import { LoadingPlaceholder } from "../LoadingPlaceholder.tsx";

export const BanEditModal = NiceModal.create(({ banId }: { banId: number }) => {
	const { data: record, isLoading, isError, error } = useQuery(get, { banId });

	const { sendFlash, sendError } = useUserFlashCtx();
	const modal = useModal();

	const mutation = useMutation(update, {
		onSuccess: async (banRecord) => {
			sendFlash("success", "Updated ban successfully");
			modal.resolve(banRecord);
			await modal.hide();
		},
		onError: sendError,
	});

	const defaultValues: Omit<UpdateRequest, "$typeName"> = {
		banType: record?.ban?.banType ?? BanType.BANNED,
		reason: record?.ban?.reason ?? BanReason.CHEATING,
		reasonText: record?.ban?.reasonText ?? "",
		note: record?.ban?.note ?? "",
		evadeOk: record?.ban?.evadeOk ?? false,
		cidr: record?.ban?.cidr ?? "",
		validUntil: record?.ban?.validUntil,
		appealState: record?.ban?.appealState ?? AppealState.OPEN_UNSPECIFIED,
		banId: banId,
	};

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate(
				create(UpdateRequestSchema, {
					banType: value.banType,
					reason: value.reason,
					reasonText: emptyOrNullString(value.reasonText) ? undefined : value.reasonText,
					note: emptyOrNullString(value.note) ? undefined : value.note,
					evadeOk: value.evadeOk,
					validUntil: value.validUntil ? value.validUntil : record?.ban?.validUntil,
					cidr: emptyOrNullString(value.cidr) ? undefined : value.cidr,
					banId: banId,
					appealState: record?.ban?.appealState,
				}),
			);
		},
		defaultValues,
	});

	const isExpired = useMemo(() => {
		if (!record?.ban?.validUntil) {
			return true;
		}
		const validUntil = timestampDate(record?.ban?.validUntil);
		return !isAfter(validUntil, new Date());
	}, [record?.ban]);

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
					Edit Ban
				</DialogTitle>

				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 12 }}>
							<Typography>{record?.ban?.targetId}</Typography>
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
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"validUntil"}
								children={(field) => {
									return <field.DateTimeField label={"New Expiration Date"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 12 }}>
							{!isExpired ? (
								<>
									<Typography>
										Expires In:{" "}
										{formatDuration(
											intervalToDuration({
												start: new Date(),
												end: record?.ban
													? timestampDate(record?.ban.validUntil as Timestamp)
													: new Date(),
											}),
										)}
									</Typography>
									<Typography>
										Expires On:{" "}
										{formatISO9075(
											record?.ban?.validUntil
												? timestampDate(record?.ban.validUntil)
												: new Date(),
										)}
									</Typography>
								</>
							) : (
								<Typography variant="h6">Ban Expired</Typography>
							)}
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
