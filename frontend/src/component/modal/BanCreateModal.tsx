import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { createConnectQueryKey, useMutation, useTransport } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import DirectionsRunIcon from "@mui/icons-material/DirectionsRun";
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
import { BanReason, BanService, BanType, CreateRequestSchema, Origin } from "../../rpc/ban/v1/ban_pb.ts";
import { create as createBan } from "../../rpc/ban/v1/ban-BanService_connectquery.ts";
import { enumValues } from "../../util/lists.ts";
import { banTypeString } from "../../util/strings.ts";
import { emptyOrNullString, zeroStringUndefined } from "../../util/types.ts";
import { MarkdownField } from "../form/field/MarkdownField.tsx";
import { Heading } from "../Heading.tsx";

type BanCreateProps = {
	// Set when creating a ban from a user report
	reportId?: number;
	steamId?: string;
	// Set when creating a user report from the stv page.
	demoId?: number;
	// An optional demotick to provide with the demoId
	demoTick?: number;
};

export const BanCreateModal = NiceModal.create(({ reportId, steamId, demoId, demoTick }: BanCreateProps) => {
	const { sendFlash, sendError } = useUserFlashCtx();
	const modal = useModal();
	const queryClient = useQueryClient();
	const transport = useTransport();
	const mutation = useMutation(createBan, {
		onSuccess: async (banRecord) => {
			sendFlash("success", `Created ban successfully #${banRecord.ban?.banId}`);
			queryClient.invalidateQueries({
				queryKey: createConnectQueryKey({
					schema: BanService.method.query,
					cardinality: "finite",
					transport,
					input: {},
				}),
			});

			modal.resolve(banRecord.ban);
			await modal.hide();
		},
		onError: sendError,
	});

	const defaultValues = {
		reportId: reportId,
		targetId: steamId ?? "",
		banType: BanType.BANNED,
		reason: BanReason.CHEATING,
		reasonText: "",
		note: "",
		evadeOk: false,
		cidr: "",
		demoId: demoId,
		demoTick: demoTick,
		origin: Origin.REPORTED,
		validUntil: new Date(),
	};

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			const updateRequest = create(CreateRequestSchema, {
				banType: value.banType,
				reason: value.reason,
				reasonText: zeroStringUndefined(value.note),
				note: zeroStringUndefined(value.note),
				evadeOk: value.evadeOk,
				validUntil: timestampFromDate(value.validUntil),
				cidr: zeroStringUndefined(value.cidr),
				origin: Origin.WEB,
				targetId: value.targetId,
				demoTick: demoTick,
				demoId: demoId,
				reportId: reportId,
			});
			if (!emptyOrNullString(value.cidr)) {
				if (!value.cidr.includes("/")) {
					value.cidr += "/32";
				}
				updateRequest.cidr = value.cidr;
			}
			mutation.mutate(updateRequest);
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
										<field.SelectBanTypeField
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
										<field.SelectBanReasonField
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
