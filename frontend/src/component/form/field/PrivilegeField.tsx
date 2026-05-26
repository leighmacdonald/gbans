import type { Privilege } from "../../../rpc/person/v1/privilege_pb";
import SelectField from "./SelectField";

export const PrivilegeField = SelectField<Privilege>;

export default PrivilegeField;
