import type { AuthType } from "../../../rpc/sourcemod/v1/sourcemod_pb";
import SelectField from "./SelectField";

export const AuthTypeField = SelectField<AuthType>;

export default AuthTypeField;
