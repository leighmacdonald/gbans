import { createConnectQueryKey, useMutation, useQuery, useTransport } from "@connectrpc/connect-query";
import CloudUploadIcon from "@mui/icons-material/CloudUpload";
import Button from "@mui/material/Button";
import Checkbox from "@mui/material/Checkbox";
import FormControl from "@mui/material/FormControl";
import FormControlLabel from "@mui/material/FormControlLabel";
import FormHelperText from "@mui/material/FormHelperText";
import Grid from "@mui/material/Grid";
import InputLabel from "@mui/material/InputLabel";
import LinearProgress from "@mui/material/LinearProgress";
import MenuItem from "@mui/material/MenuItem";
import Select, { type SelectChangeEvent } from "@mui/material/Select";
import Typography from "@mui/material/Typography";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { type ChangeEvent, type FormEvent, useCallback, useMemo, useState } from "react";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { getDemos, uploadDemo } from "../../rpc/demo/v1/demo-DemoService_connectquery.ts";
import { serversAdmin } from "../../rpc/servers/v1/servers-ServersService_connectquery.ts";
import { logErr } from "../../util/errors.ts";
import { VisuallyHiddenInput } from "../form/field/VisuallyHiddenInput";
import { SubHeading } from "../SubHeading.tsx";

const maxDemoUploadBytes = 500_000_000;

export const ManualDemoUpload = () => {
	const { sendError, sendFlash } = useUserFlashCtx();
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const transport = useTransport();
	const [serverId, setServerId] = useState<number | "">("");
	const [file, setFile] = useState<File | null>(null);
	const [force, setForce] = useState(false);
	const [inputKey, setInputKey] = useState(0);

	const { data: serversData, isLoading: serversLoading, isError: serversError } = useQuery(serversAdmin);
	const servers = useMemo(() => {
		return [...(serversData?.servers ?? [])].sort((a, b) => a.shortName.localeCompare(b.shortName));
	}, [serversData]);

	const mutation = useMutation(uploadDemo, {
		onSuccess: async (response) => {
			sendFlash("success", `Demo imported successfully (${response.demoId})`);
			await queryClient.invalidateQueries({
				queryKey: createConnectQueryKey({
					schema: getDemos,
					cardinality: "finite",
					transport,
					input: {},
				}),
			});
			setFile(null);
			setForce(false);
			setInputKey((value) => value + 1);
			await navigate({ to: "/match/$matchId", params: { matchId: response.matchId } });
		},
		onError: (error) => {
			logErr(error);
			sendError(error);
		},
	});

	const onServerChange = useCallback((event: SelectChangeEvent<number>) => {
		setServerId(Number(event.target.value));
	}, []);

	const onFileChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
		setFile(event.target.files?.[0] ?? null);
	}, []);

	const onSubmit = useCallback(
		async (event: FormEvent) => {
			event.preventDefault();
			if (mutation.isPending) {
				return;
			}
			if (serverId === "") {
				sendFlash("error", "Select the server this demo belongs to");
				return;
			}
			if (!file) {
				sendFlash("error", "Select a demo file to upload");
				return;
			}
			if (!file.name.toLowerCase().endsWith(".dem")) {
				sendFlash("error", "Only .dem files can be uploaded");
				return;
			}
			if (file.size === 0) {
				sendFlash("error", "The selected demo file is empty");
				return;
			}
			if (file.size > maxDemoUploadBytes) {
				sendFlash("error", "Demo files must be 500 MB or smaller");
				return;
			}

			let contents: Uint8Array;
			try {
				contents = new Uint8Array(await file.arrayBuffer());
			} catch (error) {
				logErr(error);
				sendFlash("error", "Unable to read the selected demo file");
				return;
			}

			mutation.mutate({
				serverId,
				filename: file.name,
				contents,
				force,
			});
		},
		[file, force, mutation, sendFlash, serverId],
	);

	return (
		<>
			<SubHeading>Manually upload a demo, import its stats, then open the resulting match.</SubHeading>
			<form onSubmit={onSubmit}>
				<Grid container spacing={2}>
					<Grid size={{ xs: 12, md: 6 }}>
						<FormControl fullWidth required disabled={mutation.isPending || serversLoading}>
							<InputLabel id="manual-demo-upload-server-label">Server</InputLabel>
							<Select
								labelId="manual-demo-upload-server-label"
								label="Server"
								value={serverId}
								onChange={onServerChange}
							>
								{servers.map((server) => (
									<MenuItem key={server.serverId} value={server.serverId}>
										{server.shortName} — {server.name}
									</MenuItem>
								))}
							</Select>
							{serversError && <FormHelperText error>Unable to load servers</FormHelperText>}
						</FormControl>
					</Grid>
					<Grid size={{ xs: 12, md: 6 }}>
						<Button
							component="label"
							variant="outlined"
							fullWidth
							startIcon={<CloudUploadIcon />}
							disabled={mutation.isPending}
						>
							{file ? file.name : "Select demo file"}
							<VisuallyHiddenInput
								key={inputKey}
								type="file"
								accept=".dem"
								onChange={onFileChange}
								disabled={mutation.isPending}
							/>
						</Button>
						{file && (
							<Typography variant="body2" color="textSecondary" sx={{ marginTop: 1 }}>
								{(file.size / 1_000_000).toFixed(1)} MB selected
							</Typography>
						)}
					</Grid>
					<Grid size={{ xs: 12 }}>
						<FormControlLabel
							control={
								<Checkbox
									checked={force}
									disabled={mutation.isPending}
									onChange={(event) => setForce(event.target.checked)}
								/>
							}
							label="Force reimport, replacing existing stats for a duplicate demo"
						/>
					</Grid>
					{mutation.isPending && (
						<Grid size={{ xs: 12 }}>
							<LinearProgress />
							<Typography variant="body2" color="textSecondary" sx={{ marginTop: 1 }}>
								Uploading, parsing, and importing the demo. Larger demos can take several minutes.
							</Typography>
						</Grid>
					)}
					<Grid size={{ xs: 12 }}>
						<Button
							type="submit"
							variant="contained"
							startIcon={<CloudUploadIcon />}
							disabled={serverId === "" || !file || mutation.isPending}
						>
							{mutation.isPending ? "Uploading and importing…" : "Upload and import demo"}
						</Button>
					</Grid>
				</Grid>
			</form>
		</>
	);
};
