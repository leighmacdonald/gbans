import { create } from "@bufbuild/protobuf";
import { createClient } from "@connectrpc/connect";
import { createConnectQueryKey } from "@connectrpc/connect-query";
import CloudUploadIcon from "@mui/icons-material/CloudUpload";
import Button from "@mui/material/Button";
import type React from "react";
import { useCallback } from "react";
import { type Asset, AssetService, CreateRequestSchema } from "../../../rpc/asset/v1/asset_pb.ts";
import { finalTransport, queryClient } from "../../../transport.ts";
import { VisuallyHiddenInput } from "../field/VisuallyHiddenInput";

type UploadButtonProps = {
	label?: string;
	onSuccess?: (file: Asset) => void;
	icon?: React.ReactNode;
};

export const UploadButton = ({ onSuccess, icon, label = "Upload files" }: UploadButtonProps) => {
	const onChange = useCallback(
		async (e: React.ChangeEvent<HTMLInputElement>) => {
			if (!e.target.files || e.target.files.length === 0) {
				return;
			}
			const file = e.target.files[0];
			const assetClient = createClient(AssetService, finalTransport);
			const resp = await queryClient.fetchQuery({
				queryKey: createConnectQueryKey({
					schema: AssetService,
					transport: finalTransport,
					cardinality: "finite",
				}),
				queryFn: async () => {
					return await assetClient.create(
						create(CreateRequestSchema, {
							name: file.name,
							contents: await file.bytes(),
						}),
					);
				},
			});
			if (resp.asset && onSuccess) {
				onSuccess(resp.asset);
			}
		},
		[onSuccess],
	);

	return (
		<Button
			component="label"
			role={undefined}
			variant="contained"
			tabIndex={-1}
			startIcon={icon ?? <CloudUploadIcon />}
		>
			{label}
			<VisuallyHiddenInput type="file" onChange={onChange} multiple accept=".png" />
		</Button>
	);
};
