import MenuItem from "@mui/material/MenuItem";
import { type BanDuration, banDurationOptions, banDurationValues } from "../../../util/ban.ts";
import SelectField from "./SelectField.tsx";

type Props = {
	label?: string;
	handleChange?: (duration: BanDuration) => void;
};

export const SelectBanDurationField = ({ label = "Ban Duration", handleChange }: Props) => {
	return (
		<SelectField<BanDuration>
			label={label}
			items={banDurationValues}
			handleChange={handleChange}
			renderItem={(duration) => {
				const option = banDurationOptions.find((candidate) => candidate.value === duration);
				return (
					<MenuItem value={duration} key={duration}>
						{option?.label ?? duration}
					</MenuItem>
				);
			}}
		/>
	);
};

export default SelectBanDurationField;
