import { z } from "zod/v4";
import { AppealStateEnum, BanReasonEnum } from "./bans.ts";

export const schemaQueryFilter = z.object({
	offset: z.number().optional(),
	limit: z.number().optional(),
	desc: z.boolean().optional(),
	query: z.string().optional(),
	order_by: z.string().optional(),
	deleted: z.boolean().optional(),
	flagged_only: z.boolean().optional(),
});

export const schemaBanQueryOpts = z.object({
	source_id: z.string().optional(),
	target_id: z.string().optional(),
	appeal_state: AppealStateEnum.optional(),
	groups_only: z.boolean().optional(),
	deleted: z.boolean().optional(),
	cidr: z.cidrv4().optional(),
	cidr_only: z.boolean().optional(),
	reason: BanReasonEnum.optional(),
	include_groups: z.boolean().optional(),
});

export type BanQueryOpts = z.infer<typeof schemaBanQueryOpts>;

export const schemaReportQueryFilter = z.object({
	deleted: z.boolean().optional(),
	source_id: z.string().optional(),
});

export type ReportQueryFilter = z.infer<typeof schemaReportQueryFilter>;

export const schemaCallbackLink = z.object({
	url: z.url(),
});

export type CallbackLink = z.infer<typeof schemaCallbackLink>;
