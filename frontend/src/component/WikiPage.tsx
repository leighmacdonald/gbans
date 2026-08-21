import { create } from "@bufbuild/protobuf";
import { useMutation } from "@connectrpc/connect-query";
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
import { Permission } from "../rpc/roles/v1/roles_pb.ts";
import { type Wiki, WikiSchema } from "../rpc/wiki/v1/wiki_pb.ts";
import { update } from "../rpc/wiki/v1/wiki-WikiService_connectquery.ts";
import { enumValues } from "../util/lists.ts";
import { ContainerWithHeaderAndButtons } from "./ContainerWithHeaderAndButtons.tsx";
import { mdEditorRef } from "./form/field/MarkdownField.tsx";
import { MarkDownRenderer } from "./MarkdownRenderer.tsx";

export const WikiPage = ({ slug = "home", page, assetURL }: { slug: string; page: Wiki; assetURL: string }) => {
	const [editMode, setEditMode] = useState<boolean>(false);
	const [currentPage, setCurrentPage] = useState<Wiki>(page);
	const { hasPermission } = useAuth();
	const { sendFlash, sendError } = useUserFlashCtx();

	const buttons = useMemo(() => {
		if (!hasPermission(Permission.WIKI_EDIT)) {
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
			await mutation.mutateAsync({
				wiki: create(WikiSchema, {
					slug,
					bodyMd: value.bodyMd,
					requiredPermission: value.requiredPermission,
				}),
			});
		},
		validators: {
			onChange: z.object({
				requiredPermission: z.enum(Permission),
				bodyMd: z.string(),
			}),
		},
		defaultValues: {
			requiredPermission: page?.requiredPermission ?? Permission.UNSPECIFIED,
			bodyMd: page?.bodyMd ?? "",
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
								name={"requiredPermission"}
								children={(field) => {
									return (
										<field.SelectPermissionsField
											label={"Permissions"}
											items={enumValues(Permission)}
											renderItem={(pl) => {
												return (
													<MenuItem value={pl} key={`pl-${pl}`}>
														{Permission[pl]}
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
								name={"bodyMd"}
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
					<MarkDownRenderer bodyMd={currentPage?.bodyMd ?? ""} assetURL={assetURL} />
				</ContainerWithHeaderAndButtons>
			</Grid>
		</Grid>
	);
};
