import type { PersonCore } from "../rpc/person/v1/person_core_pb.ts";
import type { Permission } from "../rpc/roles/v1/roles_pb.ts";

export const hasPermission = (person: PersonCore, permission: Permission): boolean =>
	person.permissions.includes(permission);
