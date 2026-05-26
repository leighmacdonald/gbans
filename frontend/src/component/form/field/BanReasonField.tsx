import type { BanReason } from "../../../rpc/ban/v1/ban_pb";
import SelectField from "./SelectField";

export const BanReasonField = SelectField<BanReason>;

export default BanReasonField;
