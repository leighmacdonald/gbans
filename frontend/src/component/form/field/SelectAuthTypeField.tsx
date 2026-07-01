import type { AuthType } from "../../../rpc/sourcemod/v1/sourcemod_pb";
import SelectField from "./SelectField";

export const SelectAuthTypeField = SelectField<AuthType>;

export default SelectAuthTypeField;
