import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableHead from "@mui/material/TableHead";
import TableRow from "@mui/material/TableRow";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useCallback, useState } from "react";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import {
	type Bucket,
	CreateBucketRequestSchema,
	DeleteBucketRequestSchema,
	SetBucketEnabledRequestSchema,
} from "../../rpc/stats/v1/stats_pb.ts";
import {
	buckets,
	createBucket,
	deleteBucket,
	setBucketEnabled,
} from "../../rpc/stats/v1/stats-StatsService_connectquery.ts";
import { ErrorDetails } from "../ErrorDetails.tsx";
import { LoadingPlaceholder } from "../LoadingPlaceholder.tsx";
import { ConfirmationModal } from "../modal/ConfirmationModal.tsx";

export const StatBucketsAdmin = () => {
	const { sendError, sendFlash } = useUserFlashCtx();
	const [newBucketName, setNewBucketName] = useState("");
	const { data, isLoading, isError, error, refetch } = useQuery(buckets);

	const createMutation = useMutation(createBucket, {
		onSuccess: async () => {
			sendFlash("success", "Stat bucket created");
			setNewBucketName("");
			await refetch();
		},
		onError: sendError,
	});

	const deleteMutation = useMutation(deleteBucket, {
		onSuccess: async () => {
			sendFlash("success", "Stat bucket deleted");
			await refetch();
		},
		onError: sendError,
	});

	const setEnabledMutation = useMutation(setBucketEnabled, {
		onSuccess: async () => {
			await refetch();
		},
		onError: async (err) => {
			sendError(err);
			await refetch();
		},
	});

	const onCreate = useCallback(async () => {
		const bucketName = newBucketName.trim();
		if (!bucketName) {
			return;
		}

		try {
			await createMutation.mutateAsync(create(CreateBucketRequestSchema, { bucketName }));
		} catch {
			// onError already surfaces the failure.
		}
	}, [createMutation, newBucketName]);

	const onDelete = useCallback(
		async (bucket: Bucket) => {
			const confirmed = (await NiceModal.show(ConfirmationModal, {
				title: `Delete stat bucket "${bucket.bucketName}"?`,
				size: "sm",
			})) as boolean;
			if (!confirmed) {
				return;
			}

			try {
				await deleteMutation.mutateAsync(
					create(DeleteBucketRequestSchema, { statsBucketId: bucket.statsBucketId }),
				);
			} catch {
				// onError already surfaces the failure (e.g. the bucket is in use).
			}
		},
		[deleteMutation],
	);

	const onToggleEnabled = useCallback(
		async (bucket: Bucket, enabled: boolean) => {
			try {
				await setEnabledMutation.mutateAsync(
					create(SetBucketEnabledRequestSchema, {
						statsBucketId: bucket.statsBucketId,
						isEnabled: enabled,
					}),
				);
			} catch {
				// onError already surfaces the failure.
			}
		},
		[setEnabledMutation],
	);

	if (isError) {
		return <ErrorDetails error={error} />;
	}
	if (isLoading) {
		return <LoadingPlaceholder />;
	}

	const canCreate = newBucketName.trim().length > 0 && !createMutation.isPending;

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<Typography variant={"h4"}>Stat Buckets</Typography>
				<Typography variant={"body2"} color={"textSecondary"}>
					Buckets group matches by game mode so their statistics are not skewed by mixing modes. New buckets
					are enabled by default. The default bucket cannot be disabled or deleted, and buckets with matches
					or servers still assigned cannot be deleted.
				</Typography>
			</Grid>

			<Grid size={{ xs: 12 }}>
				<Stack direction={{ xs: "column", sm: "row" }} spacing={1}>
					<TextField
						size={"small"}
						label={"New bucket name"}
						value={newBucketName}
						onChange={(event) => setNewBucketName(event.target.value)}
						onKeyDown={(event) => {
							if (event.key === "Enter") {
								event.preventDefault();
								void onCreate();
							}
						}}
					/>
					<Button variant={"contained"} startIcon={<AddIcon />} disabled={!canCreate} onClick={onCreate}>
						Add Bucket
					</Button>
				</Stack>
			</Grid>

			<Grid size={{ xs: 12 }}>
				<TableContainer component={Paper} variant={"outlined"}>
					<Table size={"small"}>
						<TableHead>
							<TableRow>
								<TableCell>Name</TableCell>
								<TableCell width={80}>ID</TableCell>
								<TableCell width={120}>Enabled</TableCell>
								<TableCell width={110}>Default</TableCell>
								<TableCell width={80} align={"right"}>
									Actions
								</TableCell>
							</TableRow>
						</TableHead>
						<TableBody>
							{(data?.buckets ?? []).map((bucket) => (
								<TableRow key={bucket.statsBucketId} hover>
									<TableCell>{bucket.bucketName}</TableCell>
									<TableCell>{bucket.statsBucketId}</TableCell>
									<TableCell>
										<Switch
											size={"small"}
											checked={bucket.isEnabled}
											disabled={bucket.isDefault || setEnabledMutation.isPending}
											onChange={(event) => void onToggleEnabled(bucket, event.target.checked)}
											slotProps={{
												input: {
													"aria-label": `Enable ${bucket.bucketName}`,
												},
											}}
										/>
									</TableCell>
									<TableCell>
										{bucket.isDefault ? <Chip size={"small"} label={"Default"} /> : null}
									</TableCell>
									<TableCell align={"right"}>
										<IconButton
											color={"error"}
											size={"small"}
											disabled={bucket.isDefault || deleteMutation.isPending}
											aria-label={`Delete ${bucket.bucketName}`}
											onClick={() => void onDelete(bucket)}
										>
											<DeleteIcon fontSize={"small"} />
										</IconButton>
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</TableContainer>
			</Grid>
		</Grid>
	);
};

export default StatBucketsAdmin;
