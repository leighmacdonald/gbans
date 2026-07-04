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
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { AppealState, UpdateRequestSchema } from "../rpc/ban/v1/ban_pb.ts";
import { get, update } from "../rpc/ban/v1/ban-BanService_connectquery.ts";
import { enumValues } from "../util/lists.ts";
import { toTitleCase } from "../util/strings.ts";
import { zeroStringUndefined } from "../util/types.ts";
import { ButtonLink } from "./ButtonLink.tsx";
import { ContainerWithHeader } from "./ContainerWithHeader";
import { ErrorDetails } from "./ErrorDetails.tsx";
import { LoadingPlaceholder } from "./LoadingPlaceholder.tsx";
import { BanCreateModal } from "./modal/BanCreateModal.tsx";
import { UnbanModal } from "./modal/UnbanModal.tsx";

const onSubmit = z.object({
	appealState: z.enum(AppealState),
});

export const BanModPanel = ({ banId }: { banId: number }) => {
	const navigate = useNavigate();
	const { sendFlash, sendError } = useUserFlashCtx();

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
		await NiceModal.show(BanCreateModal, {});
	}, []);

	const appealStateMutation = useMutation(update, {
		onSuccess: () => {
			sendFlash("success", "Appeal State Updated");
		},
		onError: (err) => {
			sendError(err);
		},
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			const opts = {
				banType: data?.ban?.banType,
				banId: data?.ban?.banId,
				appealState: value.appealState,
				cidr: zeroStringUndefined(data?.ban?.cidr),
				validUntil: data?.ban?.validUntil,
				evadeOk: data?.ban?.evadeOk,
				note: zeroStringUndefined(data?.ban?.note),
				reason: data?.ban?.reason,
				reasonText: zeroStringUndefined(data?.ban?.reasonText),
			};
			console.log(opts);
			appealStateMutation.mutate(create(UpdateRequestSchema, opts));
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
											<field.SelectAppealStateField
												label={"Appeal State"}
												value={field.state.value}
												items={enumValues(AppealState)}
												renderItem={(i) => {
													return (
														<MenuItem value={i} key={AppealState[i]}>
															{toTitleCase(AppealState[i])}
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
							flaggedOnly: false,
							columnFilters: [{ id: "steamId", value: data?.ban?.targetId }],
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
						search={{ columnFilters: [{ id: "targetId", value: data?.ban?.targetId }] }}
						startIcon={<NoAccountsIcon />}
					>
						Ban History
					</ButtonLink>

					<ButtonLink
						variant={"contained"}
						color={"secondary"}
						to={"/admin/reports"}
						search={{ columnFilters: [{ id: "targetId", value: data?.ban?.targetId }] }}
						startIcon={<VideocamIcon />}
					>
						Report History
					</ButtonLink>

					<ButtonLink
						variant={"contained"}
						color={"secondary"}
						to={"/admin/network/playersbyip"}
						search={{ columnFilters: [{ id: "targetId", value: data?.ban?.targetId }] }}
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
