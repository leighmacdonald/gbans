import { useQuery } from "@connectrpc/connect-query";
import { QuestionMark } from "@mui/icons-material";
import ErrorOutlineIcon from "@mui/icons-material/ErrorOutline";
import HourglassBottomIcon from "@mui/icons-material/HourglassBottom";
import Avatar from "@mui/material/Avatar";
import InputAdornment from "@mui/material/InputAdornment";
import type { TextFieldProps } from "@mui/material/TextField";
import * as MUITextField from "@mui/material/TextField";
import { useStore } from "@tanstack/react-form";
import { useAsyncDebouncedCallback } from "@tanstack/react-pacer";
import { type ChangeEvent, useCallback, useEffect, useMemo, useState } from "react";
import { useFieldContext } from "../../../contexts/formContext.tsx";
import type { ResolveSteamIDResponse } from "../../../rpc/person/v1/person_pb.ts";
import { resolveSteamID } from "../../../rpc/person/v1/person-PersonService_connectquery.ts";
import { avatarHashToURL, defaultAvatarHash } from "../../../util/strings.ts";
import { emptyOrNullString } from "../../../util/types.ts";
import { GradientSpinner } from "../../GradientSpinner.tsx";
import { renderHelpText } from "./renderHelpText.ts";

type Props = {
	defaultProfile?: ResolveSteamIDResponse;
} & TextFieldProps;

export const SteamIDField = (props: Props) => {
	const field = useFieldContext<string>();
	const errors = useStore(field.store, (state) => state.meta.errors);
	const [profile, setProfile] = useState<ResolveSteamIDResponse | undefined>(props.defaultProfile);
	const [error, setError] = useState<string>();
	const [steamId, setSteamId] = useState("");

	const { data, isLoading, isRefetching, isError } = useQuery(
		resolveSteamID,
		{ steamId },
		{ enabled: !emptyOrNullString(steamId) },
	);

	useEffect(() => {
		if (isLoading || isRefetching || !data) {
			return;
		}
		if (isError) {
			return;
		}
		setProfile(data);
		field.setValue(data.steamId.toString());
		setError(undefined);
	}, [data, isLoading, field.setValue, isRefetching, isError]);

	const debounced = useAsyncDebouncedCallback(
		async () => {
			if (!emptyOrNullString(field.state.value)) {
				setSteamId(field.state.value);
			} else {
				setProfile(undefined);
				setSteamId("");
			}
		},
		{ wait: 500 },
	);

	const adornment = useMemo(() => {
		if (isLoading || isRefetching) {
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
			return <Avatar src={avatarHashToURL(profile.avatarHash ?? defaultAvatarHash)} variant={"square"} />;
		}

		return <QuestionMark color={"secondary"} />;
	}, [
		field.state.meta.isPristine,
		field.state.meta.isValidating,
		profile,
		field.state.meta.errors,
		error,
		isLoading,
		isRefetching,
	]);

	const onChange = useCallback(
		(e: ChangeEvent<HTMLInputElement>) => {
			field.handleChange(e.target.value);
			//setProfile(undefined);

			// Trigger a debounced validation check
			debounced();
		},
		[field, debounced],
	);

	return (
		<MUITextField.default
			{...props}
			value={field.state.value}
			onChange={onChange}
			onBlur={field.handleBlur}
			fullWidth
			error={Boolean(error)}
			helperText={renderHelpText(errors, "Any form of Steam ID or profile link.")}
			slotProps={{
				input: {
					endAdornment: <InputAdornment position={"end"}>{adornment}</InputAdornment>,
				},
			}}
		/>
	);
};
