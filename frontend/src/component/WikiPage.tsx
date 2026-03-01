import ArticleIcon from "@mui/icons-material/Article";
import BuildIcon from "@mui/icons-material/Build";
import EditIcon from "@mui/icons-material/Edit";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { z } from "zod/v4";
import { apiSaveWikiPage } from "../api/wiki.ts";
import { useAppForm } from "../contexts/formContext.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import {
	PermissionLevel,
	PermissionLevelCollection,
	type PermissionLevelEnum,
	permissionLevelString,
} from "../schema/people.ts";
import type { Page } from "../schema/wiki.ts";
import { ContainerWithHeaderAndButtons } from "./ContainerWithHeaderAndButtons.tsx";
import { mdEditorRef } from "./form/field/MarkdownField.tsx";
import { MarkDownRenderer } from "./MarkdownRenderer.tsx";

interface WikiValues {
	body_md: string;
	permission_level: PermissionLevelEnum;
}

export const WikiPage = ({ slug = "home", page, assetURL }: { slug: string; page: Page; assetURL: string }) => {
	const [editMode, setEditMode] = useState<boolean>(false);
	const [currentPage, setCurrentPage] = useState<Page>(page);
	const queryClient = useQueryClient();
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
			</ButtonGroup>,
		];
	}, [hasPermission]);

	const mutation = useMutation({
		mutationKey: ["wiki", { slug }],
		mutationFn: async (values: WikiValues) => {
			const newPage: Page = {
				body_md: values.body_md,
				slug: slug ?? "home",
				permission_level: values.permission_level,
				created_on: page?.created_on ?? new Date(),
				updated_on: page?.updated_on ?? new Date(),
			};

			return await apiSaveWikiPage(newPage);
		},
		onSuccess: (savedPage) => {
			queryClient.setQueryData(["wiki", { slug }], savedPage);
			setEditMode(false);
			mdEditorRef.current?.setMarkdown("");
			sendFlash("success", `Updated ${slug} successfully. Revision: ${savedPage.revision}`);
			setCurrentPage(savedPage);
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate(value);
		},
		validators: {
			onChange: z.object({
				permission_level: z.enum(PermissionLevel),
				body_md: z.string(),
			}),
		},
		defaultValues: {
			permission_level: page?.permission_level ?? PermissionLevel.Guest,
			body_md: page?.body_md ?? "",
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
									return <field.MarkdownField label={"Region"} />;
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
					<MarkDownRenderer body_md={currentPage?.body_md ?? ""} assetURL={assetURL} />
				</ContainerWithHeaderAndButtons>
			</Grid>
		</Grid>
	);
};
