import { createFormHook, createFormHookContexts } from "@tanstack/react-form";
import { lazy } from "react";

const ClearButton = lazy(() => import("../component/form/button/ClearButton.tsx"));
const CloseButton = lazy(() => import("../component/form/button/CloseButton.tsx"));
const ResetButton = lazy(() => import("../component/form/button/ResetButton.tsx"));
const SubmitButton = lazy(() => import("../component/form/button/SubmitButton.tsx"));
const CheckboxField = lazy(() => import("../component/form/field/CheckboxField.tsx"));
const DateTimeField = lazy(() => import("../component/form/field/DateTimeField.tsx"));
const MarkdownField = lazy(() => import("../component/form/field/MarkdownField.tsx"));
const NumberField = lazy(() => import("../component/form/field/NumberField.tsx"));
const SelectField = lazy(() => import("../component/form/field/SelectField.tsx"));
const SelectBanReasonField = lazy(() => import("../component/form/field/SelectBanReasonField.tsx"));
const SelectBanTypeField = lazy(() => import("../component/form/field/SelectBanTypeField.tsx"));
const SelectAppealStateField = lazy(() => import("../component/form/field/SelectAppealStateField.tsx"));
const SelectReportStatusField = lazy(() => import("../component/form/field/SelectReportStatusField.tsx"));
const SelectDemoStrategyField = lazy(() => import("../component/form/field/SelectDemoStrategyField.tsx"));
const SelectStatsTimeBucketField = lazy(() => import("../component/form/field/SelectStatsTimeBucketField.tsx"));
const SelectStatsVariantField = lazy(() => import("../component/form/field/SelectStatsVariantField.tsx"));
const SelectLevelField = lazy(() => import("../component/form/field/SelectLevelField.tsx"));
const SelectAuthTypeField = lazy(() => import("../component/form/field/SelectAuthTypeField.tsx"));
const SelectActionField = lazy(() => import("../component/form/field/SelectActionField.tsx"));
const SelectPrivilegeField = lazy(() => import("../component/form/field/SelectPrivilegeField.tsx"));
const SelectDiscordRolesField = lazy(() => import("../component/form/field/SelectDiscordRolesField.tsx"));
const SteamIDField = lazy(() => import("../component/form/field/SteamIDField.tsx"));
const TextField = lazy(() => import("../component/form/field/TextField.tsx"));
const SelectBucketField = lazy(() => import("../component/form/field/SelectBucketField.tsx"));
const SelectSelectStringField = lazy(() => import("../component/form/field/SelectStringField.tsx"));
const SelectOverrideTypeField = lazy(() => import("../component/form/field/SelectOverrideTypeField.tsx"));
const SelectGroupField = lazy(() => import("../component/form/field/SelectGroupField.tsx"));
const SelectForumCategoryField = lazy(() => import("../component/form/field/SelectForumCategoryField.tsx"));
const SelectOverrideAccessField = lazy(() => import("../component/form/field/SelectOverrideAccessField.tsx"));

export const { fieldContext, formContext, useFieldContext, useFormContext } = createFormHookContexts();

export const { useAppForm, withForm } = createFormHook({
	fieldContext,
	formContext,
	fieldComponents: {
		CheckboxField,
		MarkdownField,
		NumberField,
		SteamIDField,
		SelectField,
		SelectBanReasonField,
		SelectBanTypeField,
		SelectAppealStateField,
		DateTimeField,
		SelectPrivilegeField,
		SelectReportStatusField,
		SelectDemoStrategyField,
		SelectLevelField,
		SelectActionField,
		SelectStatsTimeBucketField,
		SelectStatsVariantField,
		SelectDiscordRolesField,
		SelectBucketField,
		SelectAuthTypeField,
		SelectSelectStringField,
		SelectOverrideTypeField,
		SelectGroupField,
		SelectForumCategoryField,
		SelectOverrideAccessField,
		TextField,
	},
	formComponents: {
		SubmitButton,
		ClearButton,
		ResetButton,
		CloseButton,
	},
});
