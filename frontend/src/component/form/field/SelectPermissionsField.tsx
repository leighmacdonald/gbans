import type { Permission } from "../../../rpc/roles/v1/roles_pb";
import SelectField from "./SelectField";

export const SelectPermissionsField = SelectField<Permission>;

export default SelectPermissionsField;
