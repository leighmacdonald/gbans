import type { BanReason } from "../../../rpc/ban/v1/ban_pb";
import SelectField from "./SelectField";

export const SelectBanReasonField = SelectField<BanReason>;

export default SelectBanReasonField;
