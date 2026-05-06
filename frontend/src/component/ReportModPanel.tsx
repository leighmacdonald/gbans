import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import AutoFixNormalIcon from "@mui/icons-material/AutoFixNormal";
import GavelIcon from "@mui/icons-material/Gavel";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import MenuItem from "@mui/material/MenuItem";
import Stack from "@mui/material/Stack";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { useAppForm } from "../contexts/formContext.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { ReportStatus } from "../rpc/ban/v1/report_pb.ts";
import { report, reportStatusEdit } from "../rpc/ban/v1/report-ReportService_connectquery.ts";
import { enumValues } from "../util/lists.ts";
import { ContainerWithHeader } from "./ContainerWithHeader";
import { ErrorDetails } from "./ErrorDetails.tsx";
import { LoadingPlaceholder } from "./LoadingPlaceholder.tsx";
import { BanModal } from "./modal/BanModal.tsx";

export const ReportModPanel = ({ reportId }: { reportId: number }) => {
	const queryClient = useQueryClient();
	const { sendFlash, sendError } = useUserFlashCtx();
	const [status, setStatus] = useState(ReportStatus.OPENED_UNSPECIFIED);

	const { data: reportResponse, isLoading, isError, error } = useQuery(report, { reportId });

	const stateMutation = useMutation(reportStatusEdit, {
		onSuccess: async (_, reportStatus) => {
			if (!reportResponse?.report || !reportStatus.reportStatus) {
				return;
			}
			sendFlash(
				"success",
				`State changed from ${
					ReportStatus[reportResponse?.report?.report?.reportStatus ?? ReportStatus.OPENED_UNSPECIFIED]
				} => ${ReportStatus[reportStatus.reportStatus ?? ReportStatus.OPENED_UNSPECIFIED]}`,
			);
			setStatus(reportStatus.reportStatus);
		},
		onError: sendError,
	});

	const onBan = useCallback(async () => {
		if (!reportResponse?.report) {
			return;
		}

		try {
			const banRecord = await NiceModal.show(BanModal, {
				reportId: Number(reportResponse?.report.report?.reportId),
				steamId: reportResponse?.report.subject?.steamId,
			});
			queryClient.setQueryData(["ban", { targetId: reportResponse.report.report?.targetId }], banRecord);
			stateMutation.mutate({});
		} catch (e) {
			sendFlash("error", `Failed to ban: ${e}`);
		}
	}, [queryClient, sendFlash, stateMutation.mutate, reportResponse?.report]);

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			if (value.report_status === reportResponse?.report?.report?.reportStatus) {
				return;
			}
			stateMutation.mutate({ reportId, reportStatus: status });
		},

		defaultValues: {
			report_status: reportResponse?.report?.report?.reportStatus ?? ReportStatus.OPENED_UNSPECIFIED,
		},
	});

	if (isLoading) {
		return <LoadingPlaceholder />;
	}

	if (isError) {
		return <ErrorDetails error={error} />;
	}

	return (
		<form
			onSubmit={async (e) => {
				e.preventDefault();
				e.stopPropagation();
				await form.handleSubmit();
			}}
		>
			<ContainerWithHeader title={"Resolve Report"} iconLeft={<AutoFixNormalIcon />}>
				<List>
					<ListItem>
						<Stack sx={{ width: "100%" }} spacing={2}>
							<form.AppField
								name={"report_status"}
								children={(field) => {
									return (
										<field.SelectField
											label={"Report State"}
											items={enumValues(ReportStatus)}
											renderItem={(i) => {
												return (
													<MenuItem key={i} value={i}>
														{ReportStatus[i]}
													</MenuItem>
												);
											}}
										/>
									);
								}}
							/>
							<form.AppForm>
								<ButtonGroup fullWidth>
									{report && (
										<Button
											variant={"contained"}
											color={"error"}
											startIcon={<GavelIcon />}
											onClick={onBan}
										>
											Ban Player
										</Button>
									)}
									<form.SubmitButton label={"Set State"} />
								</ButtonGroup>
							</form.AppForm>
						</Stack>
					</ListItem>
				</List>
			</ContainerWithHeader>
		</form>
	);
};
