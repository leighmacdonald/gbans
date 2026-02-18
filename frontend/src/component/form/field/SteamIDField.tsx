import { QuestionMark } from "@mui/icons-material";
import ErrorOutlineIcon from "@mui/icons-material/ErrorOutline";
import HourglassBottomIcon from "@mui/icons-material/HourglassBottom";
import Avatar from "@mui/material/Avatar";
import InputAdornment from "@mui/material/InputAdornment";
import TextField, { type TextFieldProps } from "@mui/material/TextField";
import { useStore } from "@tanstack/react-form";
import { debounce } from "@tanstack/react-pacer/debouncer";
import { type ChangeEvent, useCallback, useMemo, useState } from "react";
import { apiGetSteamValidate, defaultAvatarHash } from "../../../api";
import { useFieldContext } from "../../../contexts/formContext.tsx";
import type { SteamValidate } from "../../../schema/people.ts";
import { avatarHashToURL } from "../../../util/text.tsx";
import { emptyOrNullString } from "../../../util/types.ts";
import { GradientSpinner } from "../../GradientSpinner.tsx";

type Props = {
	defaultProfile?: SteamValidate;
} & TextFieldProps;

export const SteamIDField = (props: Props) => {
	const field = useFieldContext<string>();
	const errors = useStore(field.store, (state) => state.meta.errors);
	const [profile, setProfile] = useState<SteamValidate | undefined>(props.defaultProfile);
	const [error, setError] = useState<string>();
	const [loading, setLoading] = useState(false);

	// An ugly function to validate the steamid provided.
	const check = async () => {
		if (!emptyOrNullString(field.state.value)) {
			try {
				setLoading(true);
				const update = await apiGetSteamValidate(field.state.value);
				setProfile(update);
				field.setValue(update.steam_id);
				setError(undefined);
			} catch {
				// Doesnt work?
				field.setErrorMap({
					onChange: errors.map(() => "Invalid steam ID / Profile link"),
				});
				setError("Invalid steam ID / Profile link");
				setProfile(undefined);
			} finally {
				setLoading(false);
			}
		} else {
			setProfile(undefined);
		}
	};

	// Make sure we debounce the check callback
	const setDebouncedQuery = useCallback(debounce(check, { wait: 500 }), []);

	const adornment = useMemo(() => {
		if (loading) {
			return <GradientSpinner />;
		}
		if (field.state.meta.isValidating) {
			return <HourglassBottomIcon color={"warning"} sx={{ width: 40 }} />;
		}
		if (field.state.meta.isPristine) {
			return <QuestionMark color={"secondary"} />;
		}
		if (error || field.state.meta.errors.length > 0) {
			return <ErrorOutlineIcon color={"error"} sx={{ width: 40 }} />;
		}
		if (profile) {
			return <Avatar src={avatarHashToURL(profile.hash ?? defaultAvatarHash)} variant={"square"} />;
		}

		return <QuestionMark color={"secondary"} />;
	}, [field.state.meta.isPristine, field.state.meta.isValidating, profile, field.state.meta.errors, error, loading]);

	const onChange = (e: ChangeEvent<HTMLInputElement>) => {
		// Update the value immediately
		field.handleChange(e.target.value);
		setProfile(undefined);

		// Trigger a debounced validation check
		setDebouncedQuery();
	};

	return (
		<TextField
			{...props}
			value={field.state.value}
			onChange={onChange}
			onBlur={field.handleBlur}
			color={profile ? "success" : "primary"}
			fullWidth={true}
			error={Boolean(error)}
			helperText={error ?? "Any form of Steam ID or profile link."}
			slotProps={{
				input: {
					endAdornment: <InputAdornment position={"end"}>{adornment}</InputAdornment>,
				},
			}}
		/>
	);
};
