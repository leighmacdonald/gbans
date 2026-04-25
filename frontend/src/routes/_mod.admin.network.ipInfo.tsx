import CellTowerIcon from "@mui/icons-material/CellTower";
import FilterListIcon from "@mui/icons-material/FilterList";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import Table from "@mui/material/Table";
import TableBody from "@mui/material/TableBody";
import TableCell from "@mui/material/TableCell";
import TableContainer from "@mui/material/TableContainer";
import TableRow from "@mui/material/TableRow";
import Typography from "@mui/material/Typography";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { type ReactNode, useMemo } from "react";
import { MapContainer } from "react-leaflet/MapContainer";
import { Marker } from "react-leaflet/Marker";
import { TileLayer } from "react-leaflet/TileLayer";
import "leaflet/dist/leaflet.css";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import { z } from "zod/v4";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { LoadingPlaceholder } from "../component/LoadingPlaceholder.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import { queryNetwork } from "../rpc/network/v1/network-NetworkService_connectquery.ts";
import { getFlagEmoji } from "../util/emoji.ts";

const searchSchema = z.object({
	ip: z.ipv4().optional(),
});

const schema = z.object({
	ip: z.ipv4(),
});

export const Route = createFileRoute("/_mod/admin/network/ipInfo")({
	component: AdminNetworkInfo,
	validateSearch: (search) => searchSchema.parse(search),
	head: ({ match }) => ({
		meta: [{ name: "description", content: "IP Info" }, match.context.title("IP Info")],
	}),
});

const InfoRow = ({ label, children }: { label: string; children: ReactNode }) => {
	return (
		<TableRow hover>
			<TableCell>
				<Typography fontWeight={700}> {label}</Typography>
			</TableCell>
			<TableCell>{children}</TableCell>
		</TableRow>
	);
};

function AdminNetworkInfo() {
	const navigate = useNavigate({ from: Route.fullPath });
	const { ip } = Route.useSearch();
	const { data, isLoading } = useQuery(queryNetwork, { ip });

	const defaultValues: z.input<typeof searchSchema> = {
		ip: ip ?? "",
	};

	const pos = useMemo(() => {
		if (!data || data?.details?.location?.latLong?.latitude === 0) {
			return { lat: 50, lng: 50 };
		}
		return {
			lat: data?.details?.location?.latLong?.latitude ?? 50,
			lng: data?.details?.location?.latLong?.longitude ?? 50,
		};
	}, [data]);

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			await navigate({
				to: "/admin/network/ipInfo",
				search: (prev) => ({ ...prev, ...value }),
			});
		},
		validators: {
			onChange: schema,
		},
		defaultValues,
	});

	const clear = async () => {
		await navigate({
			to: "/admin/network/ipInfo",
			search: (prev) => ({ ...prev, ip: undefined }),
		});
	};
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeader title={"Filters"} iconLeft={<FilterListIcon />} marginTop={2}>
					<form
						onSubmit={async (e) => {
							e.preventDefault();
							e.stopPropagation();
							await form.handleSubmit();
						}}
					>
						<Grid container spacing={2}>
							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"ip"}
									children={(field) => {
										return <field.TextField label={"IP Address"} />;
									}}
								/>
							</Grid>

							<Grid size={{ xs: 12 }}>
								<form.AppForm>
									<ButtonGroup>
										<form.ClearButton onClick={clear} />
										<form.ResetButton />
										<form.SubmitButton />
									</ButtonGroup>
								</form.AppForm>
							</Grid>
						</Grid>
					</form>
				</ContainerWithHeader>
			</Grid>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeader title="Network Info" iconLeft={<CellTowerIcon />}>
					<Grid container spacing={2}>
						<Grid size={{ xs: 12 }}>
							{/*<Formik onSubmit={onSubmit} initialValues={{ ip: '' }}>*/}
							<Grid container direction="row" alignItems="top" justifyContent="center" spacing={2}>
								{/*<Grid xs>*/}
								{/*    <IPField />*/}
								{/*</Grid>*/}
								{/*<Grid size={{xs: 2}}>*/}
								{/*    <SubmitButton*/}
								{/*        label={'Submit'}*/}
								{/*        fullWidth*/}
								{/*        disabled={loading}*/}
								{/*        startIcon={<SearchIcon />}*/}
								{/*    />*/}
								{/*</Grid>*/}
							</Grid>
							{/*</Formik>*/}
						</Grid>
					</Grid>
					<Grid container spacing={2}>
						<Grid size={{ xs: 12 }}>
							{isLoading ? (
								<LoadingPlaceholder />
							) : (
								<div>
									<Grid container spacing={2}>
										<Grid size={{ xs: 12, md: 6 }}>
											<Typography variant={"h4"} padding={2}>
												Location
											</Typography>
											<TableContainer>
												<Table>
													<TableBody>
														<InfoRow label={"Country"}>
															{data && (
																<>
																	{data?.details?.location?.countryCode &&
																		getFlagEmoji(
																			data?.details?.location?.countryCode,
																		)}{" "}
																	{data?.details?.location?.countryCode} (
																	{data?.details?.location?.countryCode})
																</>
															)}
														</InfoRow>
														<InfoRow label={"Region"}>
															{data?.details?.location?.regionName}
														</InfoRow>
														<InfoRow label={"City"}>
															{data?.details?.location?.cityName}
														</InfoRow>
														<InfoRow label={"Latitude"}>
															{data?.details?.location?.latLong?.latitude}
														</InfoRow>
														<InfoRow label={"Longitude"}>
															{data?.details?.location?.latLong?.longitude}
														</InfoRow>
													</TableBody>
												</Table>
											</TableContainer>
										</Grid>
										<Grid size={{ xs: 12, md: 6 }} padding={2}>
											<MapContainer
												zoom={3}
												scrollWheelZoom={true}
												id={"map"}
												style={{
													height: "400px",
													width: "100%",
												}}
												attributionControl={true}
												minZoom={3}
												worldCopyJump={true}
												center={pos}
											>
												<TileLayer
													url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
													attribution={"© OpenStreetMap contributors "}
												/>
												{(data?.details?.location?.latLong?.latitude || 0) > 0 && (
													<Marker autoPan={true} title={"Location"} position={pos} />
												)}
											</MapContainer>
										</Grid>
										<Grid size={{ xs: 12, md: 6 }}>
											<Typography variant={"h4"} padding={2}>
												ASN
											</Typography>
											<TableContainer>
												<Table>
													<TableBody>
														<InfoRow label={"AS Name"}>
															{data?.details?.asn?.asName}
														</InfoRow>
														<InfoRow label={"AS Number"}>
															<Link
																href={`https://bgpview.io/asn/${data?.details?.asn?.asNum}`}
															>
																{data?.details?.asn?.asNum}
															</Link>
														</InfoRow>
														<InfoRow label={"CIDR Block"}>
															{data?.details?.asn?.cidr}
														</InfoRow>
													</TableBody>
												</Table>
											</TableContainer>
										</Grid>
										<Grid size={{ xs: 12, md: 6 }}>
											<Typography variant={"h4"} padding={2}>
												Proxy Info
											</Typography>
											<TableContainer>
												<Table>
													<TableBody>
														<InfoRow label={"Proxy Type"}>
															{data?.details?.proxy?.proxyType}
														</InfoRow>
														<InfoRow label={"ISP"}>{data?.details?.proxy?.isp}</InfoRow>
														<InfoRow label={"Domain"}>
															{data?.details?.proxy?.domain}
														</InfoRow>
														<InfoRow label={"Usage Type"}>
															{data?.details?.proxy?.usageType}
														</InfoRow>
														{data?.details?.proxy?.lastSeen && (
															<InfoRow label={"Last Seen"}>
																{timestampDate(
																	data?.details?.proxy?.lastSeen,
																).toString()}
															</InfoRow>
														)}
														<InfoRow label={"Threat"}>
															{data?.details?.proxy?.threatType}
														</InfoRow>
													</TableBody>
												</Table>
											</TableContainer>
										</Grid>
									</Grid>
								</div>
							)}
						</Grid>
					</Grid>
				</ContainerWithHeader>
			</Grid>
		</Grid>
	);
}
