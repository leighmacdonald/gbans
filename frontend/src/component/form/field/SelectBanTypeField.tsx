import type { BanType } from "../../../rpc/ban/v1/ban_pb";
import SelectField from "./SelectField";

export const SelectBanTypeField = SelectField<BanType>;

export default SelectBanTypeField;
