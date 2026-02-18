import { z } from "zod/v4";
import { PermissionLevelEnum } from "./people.ts";

export const Operation = {
	JoinQueue: 0,
	LeaveQueue: 1,
	Message: 2,
	StateUpdate: 3,
	StartGame: 4,
	Purge: 5,
	Bye: 6,
	ChatStatusChange: 7,
} as const;

export const OperationEnum = z.nativeEnum(Operation);
export type OperationEnum = z.infer<typeof OperationEnum>;

export const schemaQueueMember = z.object({
	name: z.string(),
	steam_id: z.string(),
	hash: z.string(),
});

export type QueueMember = z.infer<typeof schemaQueueMember>;

export const schemaPurgePayload = z.object({
	message_ids: z.array(z.number()),
});

export type PurgePayload = z.infer<typeof schemaPurgePayload>;

export const schemaClientQueueState = z.object({
	steam_id: z.string(),
});

export type ClientQueueState = z.infer<typeof schemaClientQueueState>;

export const schemaLobbyState = z.object({
	server_id: z.number(),
	members: z.array(schemaClientQueueState),
});

export type LobbyState = z.infer<typeof schemaLobbyState>;

export const schemaMessageCreatePayload = z.object({
	body_md: z.string(),
});

export type MessageCreatePayload = z.infer<typeof schemaMessageCreatePayload>;

export const schemaChatLog = z.object({
	steam_id: z.string(),
	created_on: z.date(),
	personaname: z.string(),
	avatarhash: z.string(),
	permission_level: PermissionLevelEnum,
	body_md: z.string(),
	message_id: z.number(),
});
export type ChatLog = z.infer<typeof schemaChatLog>;

export const schemaMessagePayload = z.object({
	messages: z.array(schemaChatLog),
});
export type MessagePayload = z.infer<typeof schemaMessagePayload>;

export const schemaJoinPayload = z.object({
	servers: z.array(z.number()),
});

export type JoinPayload = z.infer<typeof schemaJoinPayload>;
export type LeavePayload = JoinPayload;

export const schemaMember = z.object({
	name: z.string(),
	steam_id: z.string(),
	hash: z.string(),
});
export type Member = z.infer<typeof schemaMember>;

export const schemaClientStatePayload = z.object({
	update_users: z.boolean(),
	update_servers: z.boolean(),
	lobbies: z.array(schemaLobbyState),
	users: z.array(schemaMember),
});
export type ClientStatePayload = z.infer<typeof schemaClientStatePayload>;

export const schemaLobbyServer = z.object({
	name: z.string(),
	short_name: z.string(),
	cc: z.string(),
	connect_url: z.string(),
	connect_command: z.string(),
});
export type LobbyServer = z.infer<typeof schemaLobbyServer>;

const ChatStatus = z.enum(["readwrite", "readonly", "noaccess"]);
export type ChatStatus = z.infer<typeof ChatStatus>;

export const schemaChatStatusChangePayload = z.object({
	status: ChatStatus,
	reason: z.string(),
});

export type ChatStatusChangePayload = z.infer<typeof schemaChatStatusChangePayload>;

export const schemaGameStartPayload = z.object({
	users: z.array(schemaMember),
	server: schemaLobbyServer,
});

export type GameStartPayload = z.infer<typeof schemaGameStartPayload>;
