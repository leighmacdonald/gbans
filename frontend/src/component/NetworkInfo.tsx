import { ReactNode, useState } from 'react';
import { MapContainer, Marker, TileLayer } from 'react-leaflet';
import SearchIcon from '@mui/icons-material/Search';
import Link from '@mui/material/Link';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import * as markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png';
import * as markerIcon from 'leaflet/dist/images/marker-icon.png';
import * as markerShadow from 'leaflet/dist/images/marker-shadow.png';
import 'leaflet/dist/leaflet.css';
import { useNetworkQuery } from '../hooks/useNetworkQuery.ts';
import { getFlagEmoji } from '../util/emoji.ts';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import { IPField, IPFieldProps } from './formik/IPField.tsx';
import { SubmitButton } from './modal/Buttons.tsx';

// Workaround for leaflet not loading icons properly in react
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore
delete L.Icon.Default.prototype._getIconUrl;
L.Icon.Default.mergeOptions({
    iconRetinaUrl: markerIcon2x.default,
    iconUrl: markerIcon.default,
    shadowUrl: markerShadow.default
});

const InfoRow = ({
    label,
    children
}: {
    label: string;
    children: ReactNode;
}) => {
    return (
        <TableRow hover>
            <TableCell>
                <Typography fontWeight={700}> {label}</Typography>
            </TableCell>
            <TableCell>{children}</TableCell>
        </TableRow>
    );
};

export const NetworkInfo = () => {
    const [ip, setIP] = useState('');

    const { data, loading } = useNetworkQuery({ ip: ip });

    const onSubmit = (values: IPFieldProps) => {
        setIP(values.ip);
    };

    return (
        <>
            <Grid container spacing={2}>
                <Grid xs={12}>
                    <Formik onSubmit={onSubmit} initialValues={{ ip: '' }}>
                        <Grid
                            container
                            direction="row"
                            alignItems="top"
                            justifyContent="center"
                            spacing={2}
                        >
                            <Grid xs>
                                <IPField />
                            </Grid>
                            <Grid xs={2}>
                                <SubmitButton
                                    label={'Submit'}
                                    fullWidth
                                    disabled={loading}
                                    startIcon={<SearchIcon />}
                                />
                            </Grid>
                        </Grid>
                    </Formik>
                </Grid>
            </Grid>
            <Grid container spacing={2}>
                <Grid xs={12}>
                    {loading ? (
                        <LoadingPlaceholder />
                    ) : (
                        <div>
                            <Grid container spacing={2}>
                                <Grid xs={12} md={6}>
                                    <Typography variant={'h4'} padding={2}>
                                        Location
                                    </Typography>
                                    <TableContainer>
                                        <Table>
                                            <TableBody>
                                                <InfoRow label={'Country'}>
                                                    {getFlagEmoji(
                                                        data?.location
                                                            .country_code
                                                    )}{' '}
                                                    {
                                                        data?.location
                                                            .country_code
                                                    }{' '}
                                                    (
                                                    {
                                                        data?.location
                                                            .country_code
                                                    }
                                                    )
                                                </InfoRow>
                                                <InfoRow label={'Region'}>
                                                    {data?.location.region_name}
                                                </InfoRow>
                                                <InfoRow label={'City'}>
                                                    {data?.location.city_name}
                                                </InfoRow>
                                                <InfoRow label={'Latitude'}>
                                                    {
                                                        data?.location.lat_long
                                                            .latitude
                                                    }
                                                </InfoRow>
                                                <InfoRow label={'Longitude'}>
                                                    {
                                                        data?.location.lat_long
                                                            .longitude
                                                    }
                                                </InfoRow>
                                            </TableBody>
                                        </Table>
                                    </TableContainer>
                                </Grid>
                                <Grid xs={12} md={6} padding={2}>
                                    <MapContainer
                                        zoom={3}
                                        scrollWheelZoom={true}
                                        id={'map'}
                                        style={{
                                            height: '400px',
                                            width: '100%'
                                        }}
                                        attributionControl={true}
                                        minZoom={3}
                                        worldCopyJump={true}
                                        center={{
                                            lat: data?.location.lat_long
                                                .latitude as number,
                                            lng: data?.location.lat_long
                                                .longitude as number
                                        }}
                                    >
                                        <TileLayer
                                            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                                            attribution={
                                                '© OpenStreetMap contributors '
                                            }
                                        />
                                        {(data?.location.lat_long.latitude ||
                                            0) > 0 && (
                                            <Marker
                                                autoPan={true}
                                                title={'Location'}
                                                position={{
                                                    lat: data?.location.lat_long
                                                        .latitude as number,
                                                    lng: data?.location.lat_long
                                                        .longitude as number
                                                }}
                                            />
                                        )}
                                    </MapContainer>
                                </Grid>
                                <Grid xs={12} md={6}>
                                    <Typography variant={'h4'} padding={2}>
                                        ASN
                                    </Typography>
                                    <TableContainer>
                                        <Table>
                                            <TableBody>
                                                <InfoRow label={'AS Name'}>
                                                    {data?.asn.as_name}
                                                </InfoRow>
                                                <InfoRow label={'AS Number'}>
                                                    <Link
                                                        href={`https://bgpview.io/asn/${data?.asn.as_num}`}
                                                    >
                                                        {data?.asn.as_num}
                                                    </Link>
                                                </InfoRow>
                                                <InfoRow label={'CIDR Block'}>
                                                    {data?.asn.cidr}
                                                </InfoRow>
                                            </TableBody>
                                        </Table>
                                    </TableContainer>
                                </Grid>
                                <Grid xs={12} md={6}>
                                    <Typography variant={'h4'} padding={2}>
                                        Proxy Info
                                    </Typography>
                                    <TableContainer>
                                        <Table>
                                            <TableBody>
                                                <InfoRow label={'Proxy Type'}>
                                                    {data?.proxy.proxy_type}
                                                </InfoRow>
                                                <InfoRow label={'ISP'}>
                                                    {data?.proxy.isp}
                                                </InfoRow>
                                                <InfoRow label={'Domain'}>
                                                    {data?.proxy.domain}
                                                </InfoRow>
                                                <InfoRow label={'Usage Type'}>
                                                    {data?.proxy.usage_type}
                                                </InfoRow>
                                                <InfoRow label={'Last Seen'}>
                                                    {data?.proxy.last_seen}
                                                </InfoRow>
                                                <InfoRow label={'Threat'}>
                                                    {data?.proxy.threat}
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
        </>
    );
};
