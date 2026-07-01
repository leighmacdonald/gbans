import type { Role } from "../../../rpc/discord/v1/discord_pb";
import SelectField from "./SelectField";

export const DiscordRolesField = SelectField<Role>;

export default DiscordRolesField;
