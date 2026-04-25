import { create } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import AddModeratorIcon from "@mui/icons-material/AddModerator";
import ChatIcon from "@mui/icons-material/Chat";
import EditIcon from "@mui/icons-material/Edit";
import NoAccountsIcon from "@mui/icons-material/NoAccounts";
import UndoIcon from "@mui/icons-material/Undo";
import VideocamIcon from "@mui/icons-material/Videocam";
import WifiFindIcon from "@mui/icons-material/WifiFind";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import MenuItem from "@mui/material/MenuItem";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useNavigate } from "@tanstack/react-router";
import { useCallback, useMemo } from "react";
import z from "zod/v4";
import { useAppForm } from "../contexts/formContext.tsx";
import { AppealState, UpdateRequestSchema } from "../rpc/ban/v1/ban_pb.ts";
import { get, update } from "../rpc/ban/v1/ban-BanService_connectquery.ts";
import { enumValues } from "../util/lists.ts";
import { ButtonLink } from "./ButtonLink.tsx";
import { ContainerWithHeader } from "./ContainerWithHeader";
import { ErrorDetails } from "./ErrorDetails.tsx";
import { LoadingPlaceholder } from "./LoadingPlaceholder.tsx";
import { BanModal } from "./modal/BanModal.tsx";
import { UnbanModal } from "./modal/UnbanModal.tsx";

const onSubmit = z.object({
	appealState: z.enum(AppealState),
});

export const BanModPanel = ({ banId }: { banId: number }) => {
	const navigate = useNavigate();

	const { data, isLoading, isError, error } = useQuery(get, { banId });

	const enabled = useMemo(() => {
		if (!data?.ban?.validUntil) {
			return false;
		}

		return data.ban.validUntil ? timestampDate(data.ban.validUntil) < new Date() : false;
	}, [data?.ban?.validUntil]);

	const onUnban = useCallback(async () => {
		await NiceModal.show(UnbanModal, {
			banId,
			personaName: data?.ban?.targetPersonaName,
		});
	}, [banId, data?.ban?.targetPersonaName]);

	const onEditBan = useCallback(async () => {
		await NiceModal.show(BanModal, {
			banId,
		});
	}, [banId]);

	const appealStateMutation = useMutation(update);
	// 	mutationFn: async (appeal_state: AppealStateEnum) => {
	// 		try {
	// 			const ac = new AbortController();
	// 			await apiSetBanAppealState(ban_id, appeal_state, ac.signal);
	// 			sendFlash("success", "Appeal state updated");
	// 		} catch (reason) {
	// 			sendFlash("error", "Could not set appeal state");
	// 			logErr(reason);
	// 		}
	// 	},
	// });

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			if (!data?.ban || value.appealState === data?.ban?.appealState) {
				return;
			}

			const ban = data.ban;
			ban.appealState = value.appealState;
			appealStateMutation.mutate(
				create(UpdateRequestSchema, {
					banType: ban.banType,
					banId: ban.banId,
					appealState: ban.appealState,
					cidr: ban.cidr,
					duration: ban.duration,
					evadeOk: ban.evadeOk,
					note: ban.note,
					reason: ban.reason,
					reasonText: ban.reasonText,
				}),
			);
		},
		validators: { onSubmit },
		defaultValues: { appealState: data?.ban?.appealState ?? AppealState.OPEN_UNSPECIFIED },
	});

	if (isLoading) {
		return <LoadingPlaceholder />;
	}

	if (isError) {
		return <ErrorDetails error={error} />;
	}

	return (
		<ContainerWithHeader title={"Moderation Tools"} iconLeft={<AddModeratorIcon />}>
			<form
				onSubmit={async (e) => {
					e.preventDefault();
					e.stopPropagation();
					await form.handleSubmit();
				}}
			>
				<Stack spacing={2} padding={2}>
					<Stack direction={"row"} spacing={2}>
						{!enabled ? (
							<>
								<form.AppField
									name={"appealState"}
									children={(field) => {
										return (
											<field.SelectField
												label={"Appeal State"}
												value={field.state.value}
												items={enumValues(AppealState)}
												renderItem={(i) => {
													return (
														<MenuItem value={i} key={i}>
															{AppealState[i]}
														</MenuItem>
													);
												}}
											/>
										);
									}}
								/>
								<form.AppForm>
									<form.SubmitButton label={"Save"} />
								</form.AppForm>
							</>
						) : (
							<Typography variant={"h6"} textAlign={"center"}>
								Ban Expired
							</Typography>
						)}
					</Stack>

					{Boolean(data?.ban?.reportId) && (
						<Button
							fullWidth
							disabled={!enabled}
							color={"secondary"}
							variant={"contained"}
							onClick={async () => {
								await navigate({ to: `/report/${data?.ban?.reportId}` });
							}}
						>
							View Report #{data?.ban?.reportId}
						</Button>
					)}
					<ButtonLink
						variant={"contained"}
						color={"secondary"}
						to={"/chatlogs"}
						search={{
							flagged_only: false,
							columnFilters: [{ id: "steam_id", value: data?.ban?.targetId }],
						}}
						startIcon={<ChatIcon />}
					>
						Chat Logs
					</ButtonLink>
					<ButtonLink
						variant={"contained"}
						color={"secondary"}
						to={"/stv"}
						search={{ columnFilters: [{ id: "stats", value: data?.ban?.targetId }] }}
						startIcon={<VideocamIcon />}
					>
						STV History
					</ButtonLink>
					<ButtonLink
						variant={"contained"}
						color={"secondary"}
						to={"/admin/bans"}
						search={{ columnFilters: [{ id: "target_id", value: data?.ban?.targetId }] }}
						startIcon={<NoAccountsIcon />}
					>
						Ban History
					</ButtonLink>

					<ButtonLink
						variant={"contained"}
						color={"secondary"}
						to={"/admin/reports"}
						search={{ columnFilters: [{ id: "target_id", value: data?.ban?.targetId }] }}
						startIcon={<VideocamIcon />}
					>
						Report History
					</ButtonLink>

					<ButtonLink
						variant={"contained"}
						color={"secondary"}
						to={"/admin/network/playersbyip"}
						search={{ columnFilters: [{ id: "target_id", value: data?.ban?.targetId }] }}
						startIcon={<WifiFindIcon />}
					>
						Connection History
					</ButtonLink>

					<ButtonGroup fullWidth variant={"contained"}>
						<Button color={"warning"} onClick={onEditBan} startIcon={<EditIcon />}>
							Edit Ban
						</Button>
						<Button color={"success"} onClick={onUnban} startIcon={<UndoIcon />}>
							Unban
						</Button>
					</ButtonGroup>
				</Stack>
			</form>
		</ContainerWithHeader>
	);
};
