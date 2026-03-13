import Grid from "@mui/material/Grid";
import { createFileRoute } from "@tanstack/react-router";
import { IPWhitelistTable } from "../component/table/IPWhitelistTable.tsx";
import { NetworkBlocklist } from "../component/table/NetworkBlocklist.tsx";
import { SteamWhitelistTable } from "../component/table/SteamWhitelistTable.tsx";

export const Route = createFileRoute("/_mod/admin/network/cidrblocks")({
	component: AdminNetworkCIDRBlocks,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "CIDR Network Blocks" }, match.context.title("CIDR Network Blocks")],
	}),
});

function AdminNetworkCIDRBlocks() {
	return (
		<Grid container spacing={1}>
			<Grid size={{ xs: 12 }}>
				<NetworkBlocklist />
			</Grid>

			<Grid size={{ xs: 12 }}>
				<IPWhitelistTable />
			</Grid>

			<Grid size={{ xs: 12 }}>
				<SteamWhitelistTable />
			</Grid>
		</Grid>
	);
}
