import ArticleIcon from "@mui/icons-material/Article";
import BuildIcon from "@mui/icons-material/Build";
import EditIcon from "@mui/icons-material/Edit";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { useMemo, useState } from "react";
import { z } from "zod/v4";
import { useAppForm } from "../contexts/formContext.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import {
	PermissionLevel,
	PermissionLevelCollection,
	permissionLevelString,
} from "../schema/people.ts";
import { ContainerWithHeaderAndButtons } from "./ContainerWithHeaderAndButtons.tsx";
import { mdEditorRef } from "./form/field/MarkdownField.tsx";
import { MarkDownRenderer } from "./MarkdownRenderer.tsx";
import type {Wiki} from "../rpc/wiki/v1/wiki_pb.ts";
import  {update} from "../rpc/wiki/v1/wiki-WikiService_connectquery.ts";
import {useMutation} from "@connectrpc/connect-query";

export const WikiPage = ({ slug = "home", page, assetURL }: { slug: string; page: Wiki; assetURL: string }) => {
	const [editMode, setEditMode] = useState<boolean>(false);
	const [currentPage, setCurrentPage] = useState<Wiki>(page);
	const { hasPermission } = useAuth();
	const { sendFlash, sendError } = useUserFlashCtx();

	const buttons = useMemo(() => {
		if (!hasPermission(PermissionLevel.Editor)) {
			return [];
		}
		return [
			<ButtonGroup key={`wiki-buttons`}>
				<Button
					startIcon={<BuildIcon />}
					variant={"contained"}
					color={"warning"}
					onClick={() => {
						setEditMode(true);
					}}
				>
					Edit
				</Button>
			</ButtonGroup>
		];
	}, [hasPermission]);

	const mutation = useMutation(update, {
		onSuccess: (savedPage) => {
			//queryClient.setQueryData(["wiki", { slug }], savedPage);
			setEditMode(false);
			mdEditorRef.current?.setMarkdown("");
            if (!savedPage.wiki) {
                return;
            }
			sendFlash("success", `Updated ${slug} successfully. Revision: ${savedPage.wiki.revision}`);
			setCurrentPage(savedPage.wiki);
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			await mutation.mutateAsync({ wiki: value});
		},
		validators: {
			onChange: z.object({
				permission_level: z.enum(PermissionLevel),
				body_md: z.string(),
			}),
		},
		defaultValues: {
			permission_level: page?.permissionLevel ?? PermissionLevel.Guest,
			body_md: page?.bodyMd ?? "",
		},
	});

	if (editMode) {
		return (
			<ContainerWithHeaderAndButtons title={`Editing: ${slug}`} iconLeft={<EditIcon />}>
				<form
					onSubmit={async (e) => {
						e.preventDefault();
						e.stopPropagation();
						await form.handleSubmit();
					}}
				>
					<Grid container spacing={2}>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"permission_level"}
								children={(field) => {
									return (
										<field.SelectField
											label={"Permissions"}
											items={PermissionLevelCollection}
											renderItem={(pl) => {
												return (
													<MenuItem value={pl} key={`pl-${pl}`}>
														{permissionLevelString(pl)}
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
								name={"body_md"}
								children={(field) => {
									return <field.MarkdownField label={"Body"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 12 }}>
							<form.AppForm>
								<ButtonGroup>
									<form.CloseButton />
									<form.ResetButton />
									<form.SubmitButton />
								</ButtonGroup>
							</form.AppForm>
						</Grid>
					</Grid>
				</form>
			</ContainerWithHeaderAndButtons>
		);
	}
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: editMode ? 6 : 12 }}>
				<ContainerWithHeaderAndButtons
					title={currentPage?.slug ?? ""}
					iconLeft={<ArticleIcon />}
					buttons={buttons}
				>
					<MarkDownRenderer body_md={currentPage?.bodyMd ?? ""} assetURL={assetURL} />
				</ContainerWithHeaderAndButtons>
			</Grid>
		</Grid>
	);
};
