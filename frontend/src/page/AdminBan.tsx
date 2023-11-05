import React, { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import NiceModal from '@ebay/nice-modal-react';
import DirectionsRunIcon from '@mui/icons-material/DirectionsRun';
import EditIcon from '@mui/icons-material/Edit';
import GavelIcon from '@mui/icons-material/Gavel';
import GroupsIcon from '@mui/icons-material/Groups';
import LanIcon from '@mui/icons-material/Lan';
import RouterIcon from '@mui/icons-material/Router';
import UndoIcon from '@mui/icons-material/Undo';
import VisibilityIcon from '@mui/icons-material/Visibility';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { formatDuration, intervalToDuration } from 'date-fns';
import format from 'date-fns/format';
import {
    apiGetBansSteam,
    apiGetBansASN,
    apiGetBansCIDR,
    apiGetBansGroups,
    BanReason,
    IAPIBanASNRecord,
    IAPIBanCIDRRecord,
    IAPIBanGroupRecord,
    IAPIBanRecordProfile,
    IAPIBanRecord
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable, RowsPerPage } from '../component/DataTable';
import {
    DataTableRelativeDateField,
    isPermanentBan
} from '../component/DataTableRelativeDateField';
import { PersonCell } from '../component/PersonCell';
import { TabPanel } from '../component/TabPanel';
import {
    ModalBanASN,
    ModalBanCIDR,
    ModalBanGroup,
    ModalBanSteam,
    ModalUnbanASN,
    ModalUnbanCIDR,
    ModalUnbanGroup,
    ModalUnbanSteam
} from '../component/modal';
import { BanASNModalProps } from '../component/modal/BanASNModal';
import { BanCIDRModalProps } from '../component/modal/BanCIDRModal';
import { BanGroupModalProps } from '../component/modal/BanGroupModal';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { steamIdQueryValue } from '../util/text';

export const AdminBan = () => {
    const theme = useTheme();
    const [bans, setBans] = useState<IAPIBanRecordProfile[]>([]);
    const [banGroups, setBanGroups] = useState<IAPIBanGroupRecord[]>([]);
    const [banCIDRs, setBanCIDRs] = useState<IAPIBanCIDRRecord[]>([]);
    const [banASNs, setBanASNs] = useState<IAPIBanASNRecord[]>([]);
    const [value, setValue] = React.useState<number>(0);
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate();

    const loadBansGroup = useCallback((abortController: AbortController) => {
        apiGetBansGroups(
            { desc: true, order_by: 'ban_group_id' },
            abortController
        )
            .then((newGroupBans) => {
                setBanGroups(newGroupBans);
            })
            .catch(logErr);
    }, []);

    const loadBansCIDR = useCallback((abortController: AbortController) => {
        apiGetBansCIDR({ desc: true, order_by: 'net_id' }, abortController)
            .then((newBansCIDR) => {
                setBanCIDRs(newBansCIDR);
            })
            .catch(logErr);
    }, []);

    const loadBansASN = useCallback((abortController: AbortController) => {
        apiGetBansASN({ desc: true, order_by: 'ban_asn_id' }, abortController)
            .then((newBansASN) => {
                setBanASNs(newBansASN);
            })
            .catch(logErr);
    }, []);

    const loadBansSteam = useCallback((abortController: AbortController) => {
        apiGetBansSteam({ desc: true, order_by: 'ban_id' }, abortController)
            .then((newBans) => {
                setBans(newBans || []);
            })
            .catch(logErr);
    }, []);

    const onUnbanSteam = useCallback(
        async (ban: IAPIBanRecordProfile) => {
            try {
                await NiceModal.show(ModalUnbanSteam, {
                    banId: ban.ban_id,
                    personaName: ban.personaname
                });
                sendFlash('success', 'Unbanned successfully');
            } catch (e) {
                sendFlash('error', `Failed to unban: ${e}`);
            }
        },
        [sendFlash]
    );

    const onEditSteam = useCallback(
        async (ban: IAPIBanRecordProfile) => {
            try {
                await NiceModal.show(ModalBanSteam, {
                    banId: ban.ban_id,
                    personaName: ban.personaname,
                    existing: ban
                });
                sendFlash('success', 'Updated ban successfully');
            } catch (e) {
                sendFlash('error', `Failed to update ban: ${e}`);
            }
        },
        [sendFlash]
    );

    const onEditCIDR = useCallback(
        async (existing: IAPIBanCIDRRecord) => {
            try {
                await NiceModal.show<IAPIBanCIDRRecord, BanCIDRModalProps>(
                    ModalBanCIDR,
                    {
                        existing
                    }
                );
                sendFlash('success', 'Updated CIDR ban successfully');
            } catch (e) {
                sendFlash('error', `Failed to update ban: ${e}`);
            }
        },
        [sendFlash]
    );

    const onUnbanCIDR = useCallback(
        async (net_id: number) => {
            try {
                await NiceModal.show(ModalUnbanCIDR, {
                    banId: net_id
                });
                sendFlash('success', 'Unbanned CIDR successfully');
            } catch (e) {
                sendFlash('error', `Failed to unban: ${e}`);
            }
        },
        [sendFlash]
    );

    const onUnbanASN = useCallback(
        async (as_num: number) => {
            try {
                await NiceModal.show(ModalUnbanASN, {
                    banId: as_num
                });
                sendFlash('success', 'Unbanned ASN successfully');
            } catch (e) {
                sendFlash('error', `Failed to unban ASN: ${e}`);
            }
        },
        [sendFlash]
    );

    const onEditASN = useCallback(
        async (existing: IAPIBanASNRecord) => {
            try {
                await NiceModal.show<IAPIBanASNRecord, BanASNModalProps>(
                    ModalBanASN,
                    {
                        existing
                    }
                );
                sendFlash('success', 'Updated ASN ban successfully');
            } catch (e) {
                sendFlash('error', `Failed to update ASN ban: ${e}`);
            }
        },
        [sendFlash]
    );

    const onEditGroup = useCallback(
        async (existing: IAPIBanGroupRecord) => {
            try {
                await NiceModal.show<IAPIBanGroupRecord, BanGroupModalProps>(
                    ModalBanGroup,
                    {
                        existing
                    }
                );
                sendFlash('success', 'Updated steam group ban successfully');
            } catch (e) {
                sendFlash('error', `Failed to update steam group ban: ${e}`);
            }
        },
        [sendFlash]
    );

    const onUnbanGroup = useCallback(
        async (ban_group_id: number) => {
            try {
                await NiceModal.show(ModalUnbanGroup, {
                    banId: ban_group_id
                });
                sendFlash('success', 'Unbanned Group successfully');
            } catch (e) {
                sendFlash('error', `Failed to unban Group: ${e}`);
            }
        },
        [sendFlash]
    );

    const onNewBanSteam = useCallback(async () => {
        try {
            const ban = await NiceModal.show<IAPIBanRecord>(ModalBanSteam, {});
            sendFlash(
                'success',
                `Created steam ban successfully #${ban.ban_id}`
            );
        } catch (e) {
            sendFlash('error', `Failed to save ban: ${e}`);
        }
    }, [sendFlash]);

    const onNewBanCIDR = useCallback(async () => {
        try {
            const ban = await NiceModal.show<IAPIBanCIDRRecord>(
                ModalBanCIDR,
                {}
            );
            sendFlash(
                'success',
                `Created CIDR ban successfully #${ban.net_id}`
            );
        } catch (e) {
            sendFlash('error', `Failed to save CIDR ban: ${e}`);
        }
    }, [sendFlash]);

    const onNewBanASN = useCallback(async () => {
        try {
            const ban = await NiceModal.show<IAPIBanASNRecord>(ModalBanASN, {});
            sendFlash(
                'success',
                `Created ASN ban successfully #${ban.ban_asn_id}`
            );
        } catch (e) {
            sendFlash('error', `Failed to save ASN ban: ${e}`);
        }
    }, [sendFlash]);

    const onNewBanGroup = useCallback(async () => {
        try {
            const ban = await NiceModal.show<IAPIBanGroupRecord>(
                ModalBanGroup,
                {}
            );
            sendFlash(
                'success',
                `Created steam group ban successfully #${ban.ban_group_id}`
            );
        } catch (e) {
            sendFlash('error', `Failed to save group ban: ${e}`);
        }
    }, [sendFlash]);

    useEffect(() => {
        const abortController = new AbortController();

        loadBansSteam(abortController);
        loadBansCIDR(abortController);
        loadBansASN(abortController);
        loadBansGroup(abortController);

        return () => abortController.abort();
    }, [loadBansASN, loadBansCIDR, loadBansGroup, loadBansSteam]);

    return (
        <Grid container>
            <Grid xs={12} marginBottom={2}>
                <Box>
                    <ButtonGroup>
                        <Button
                            variant={'contained'}
                            color={'secondary'}
                            startIcon={<DirectionsRunIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanSteam}
                        >
                            Steam
                        </Button>
                        <Button
                            variant={'contained'}
                            color={'secondary'}
                            startIcon={<RouterIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanCIDR}
                        >
                            CIDR
                        </Button>
                        <Button
                            variant={'contained'}
                            color={'secondary'}
                            startIcon={<LanIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanASN}
                        >
                            ASN
                        </Button>
                        <Button
                            variant={'contained'}
                            color={'secondary'}
                            startIcon={<GroupsIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanGroup}
                        >
                            Group
                        </Button>
                    </ButtonGroup>
                </Box>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader
                    title={'Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                >
                    <Paper>
                        <Box
                            sx={{
                                borderBottom: 1,
                                borderColor: 'divider',
                                backgroundColor: theme.palette.background.paper
                            }}
                        >
                            <Tabs
                                value={value}
                                onChange={(
                                    _: React.SyntheticEvent,
                                    newValue: number
                                ) => {
                                    setValue(newValue);
                                }}
                                aria-label="ReportCreatePage detail tabs"
                            >
                                <Tab label={'Steam Bans'} color={'text'} />
                                <Tab label={`CIDR Bans`} />
                                <Tab label={`ASN Bans`} />
                                <Tab label={`Group Bans`} />
                            </Tabs>
                        </Box>

                        <TabPanel value={value} index={0}>
                            <Grid container spacing={3}>
                                <Grid xs={12}>
                                    <DataTable<IAPIBanRecordProfile>
                                        columns={[
                                            {
                                                label: '#',
                                                tooltip: 'Ban ID',
                                                sortKey: 'ban_id',
                                                sortable: true,
                                                align: 'left',
                                                queryValue: (o) =>
                                                    `${o.ban_id}`,
                                                renderer: (obj) => (
                                                    <Typography
                                                        variant={'body1'}
                                                    >
                                                        #{obj.ban_id}
                                                    </Typography>
                                                )
                                            },
                                            {
                                                label: 'Name',
                                                tooltip: 'Persona Name',
                                                sortKey: 'personaname',
                                                sortable: true,
                                                align: 'left',
                                                queryValue: (o) =>
                                                    `${o.personaname}-` +
                                                    steamIdQueryValue(
                                                        o.target_id
                                                    ),
                                                renderer: (row) => (
                                                    <PersonCell
                                                        steam_id={row.target_id}
                                                        personaname={
                                                            row.personaname
                                                        }
                                                        avatar_hash={row.avatar}
                                                    />
                                                )
                                            },
                                            {
                                                label: 'Reason',
                                                tooltip: 'Reason',
                                                sortKey: 'reason',
                                                sortable: true,
                                                align: 'left',
                                                queryValue: (o) =>
                                                    BanReason[o.reason],
                                                renderer: (row) => (
                                                    <Typography
                                                        variant={'body1'}
                                                    >
                                                        {BanReason[row.reason]}
                                                    </Typography>
                                                )
                                            },
                                            {
                                                label: 'Custom Reason',
                                                tooltip: 'Custom',
                                                sortKey: 'reason_text',
                                                sortable: false,
                                                align: 'left'
                                            },
                                            {
                                                label: 'Created',
                                                tooltip: 'Created On',
                                                sortType: 'date',
                                                align: 'left',
                                                width: '150px',
                                                virtual: true,
                                                virtualKey: 'created_on',
                                                renderer: (obj) => {
                                                    return (
                                                        <Typography
                                                            variant={'body1'}
                                                        >
                                                            {format(
                                                                obj.created_on,
                                                                'yyyy-MM-dd'
                                                            )}
                                                        </Typography>
                                                    );
                                                }
                                            },
                                            {
                                                label: 'Expires',
                                                tooltip: 'Valid Until',
                                                sortType: 'date',
                                                align: 'left',
                                                width: '150px',
                                                virtual: true,
                                                virtualKey: 'valid_until',
                                                sortable: true,
                                                renderer: (obj) => {
                                                    return (
                                                        <DataTableRelativeDateField
                                                            date={
                                                                obj.valid_until
                                                            }
                                                        />
                                                    );
                                                }
                                            },
                                            {
                                                label: 'Duration',
                                                tooltip: 'Total Ban Duration',
                                                sortType: 'number',
                                                align: 'left',
                                                width: '150px',
                                                virtual: true,
                                                virtualKey: 'duration',
                                                renderer: (row) => {
                                                    return isPermanentBan(
                                                        row.created_on,
                                                        row.valid_until
                                                    ) ? (
                                                        'Permanent'
                                                    ) : (
                                                        <DataTableRelativeDateField
                                                            date={
                                                                row.created_on
                                                            }
                                                            compareDate={
                                                                row.valid_until
                                                            }
                                                        />
                                                    );
                                                }
                                            },
                                            {
                                                label: 'Friends Incl.',
                                                tooltip:
                                                    'Are friends also included in the ban',
                                                align: 'left',
                                                width: '150px',
                                                sortKey: 'include_friends',
                                                renderer: (row) => {
                                                    return (
                                                        <Typography
                                                            variant={'body1'}
                                                        >
                                                            {row.include_friends
                                                                ? 'yes'
                                                                : 'no'}
                                                        </Typography>
                                                    );
                                                }
                                            },
                                            {
                                                label: 'Rep.',
                                                tooltip: 'Report',
                                                sortable: false,
                                                align: 'left',
                                                width: '20px',
                                                queryValue: (o) =>
                                                    `${o.report_id}`,
                                                renderer: (row) =>
                                                    row.report_id > 0 ? (
                                                        <Tooltip
                                                            title={
                                                                'View Report'
                                                            }
                                                        >
                                                            <Button
                                                                variant={'text'}
                                                                onClick={() => {
                                                                    navigate(
                                                                        `/report/${row.report_id}`
                                                                    );
                                                                }}
                                                            >
                                                                #{row.report_id}
                                                            </Button>
                                                        </Tooltip>
                                                    ) : (
                                                        <></>
                                                    )
                                            },
                                            {
                                                label: 'Act.',
                                                tooltip: 'Actions',
                                                sortKey: 'reason',
                                                sortable: false,
                                                align: 'left',
                                                renderer: (row) => (
                                                    <ButtonGroup fullWidth>
                                                        <IconButton
                                                            color={'primary'}
                                                            onClick={() => {
                                                                navigate(
                                                                    `/ban/${row.ban_id}`
                                                                );
                                                            }}
                                                        >
                                                            <Tooltip
                                                                title={'View'}
                                                            >
                                                                <VisibilityIcon />
                                                            </Tooltip>
                                                        </IconButton>
                                                        <IconButton
                                                            color={'warning'}
                                                            onClick={async () => {
                                                                await onEditSteam(
                                                                    row
                                                                );
                                                            }}
                                                        >
                                                            <Tooltip
                                                                title={
                                                                    'Edit Ban'
                                                                }
                                                            >
                                                                <EditIcon />
                                                            </Tooltip>
                                                        </IconButton>
                                                        <IconButton
                                                            color={'success'}
                                                            onClick={async () => {
                                                                await onUnbanSteam(
                                                                    row
                                                                );
                                                            }}
                                                        >
                                                            <Tooltip
                                                                title={
                                                                    'Remove Ban'
                                                                }
                                                            >
                                                                <UndoIcon />
                                                            </Tooltip>
                                                        </IconButton>
                                                    </ButtonGroup>
                                                )
                                            }
                                        ]}
                                        defaultSortColumn={'ban_id'}
                                        rowsPerPage={RowsPerPage.TwentyFive}
                                        rows={bans}
                                    />
                                </Grid>
                            </Grid>
                        </TabPanel>
                        <TabPanel value={value} index={1}>
                            <DataTable<IAPIBanCIDRRecord>
                                columns={[
                                    {
                                        label: '#',
                                        tooltip: 'Ban ID',
                                        sortKey: 'net_id',
                                        sortable: true,
                                        align: 'left',
                                        queryValue: (o) => `${o.net_id}`,
                                        renderer: (obj) => (
                                            <Typography variant={'body1'}>
                                                #{obj.net_id.toString()}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'Author',
                                        tooltip: 'Author ID',
                                        sortKey: 'source_id',
                                        sortable: true,
                                        align: 'left',
                                        queryValue: (o) =>
                                            steamIdQueryValue(o.source_id),
                                        renderer: (obj) => (
                                            <Typography variant={'body1'}>
                                                {obj.source_id.toString()}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'Target',
                                        tooltip: 'Target SID',
                                        sortKey: 'target_id',
                                        sortable: true,
                                        align: 'left',
                                        queryValue: (o) =>
                                            steamIdQueryValue(o.target_id),
                                        renderer: (obj) => (
                                            <Typography variant={'body1'}>
                                                {obj.target_id.toString()}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'CIDR',
                                        tooltip: 'CIDR Range',
                                        sortKey: 'cidr',
                                        sortable: true,
                                        align: 'left',
                                        queryValue: (o) => `${o.target_id}`,
                                        renderer: (obj) => {
                                            try {
                                                return (
                                                    <Typography
                                                        variant={'body1'}
                                                    >
                                                        {obj.cidr.IP}
                                                    </Typography>
                                                );
                                            } catch (e) {
                                                return <>?</>;
                                            }
                                        }
                                    },
                                    {
                                        label: 'Reason',
                                        tooltip: 'Reason',
                                        sortKey: 'reason',
                                        sortable: true,
                                        align: 'left',
                                        queryValue: (o) => BanReason[o.reason],
                                        renderer: (row) => (
                                            <Typography variant={'body1'}>
                                                {BanReason[row.reason]}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'Custom Reason',
                                        tooltip: 'Custom',
                                        sortKey: 'reason_text',
                                        sortable: false,
                                        align: 'left',
                                        queryValue: (o) => o.reason_text
                                    },
                                    {
                                        label: 'Created',
                                        tooltip: 'Created On',
                                        sortType: 'date',
                                        align: 'left',
                                        width: '150px',
                                        virtual: true,
                                        virtualKey: 'created_on',
                                        renderer: (obj) => {
                                            return (
                                                <DataTableRelativeDateField
                                                    date={obj.created_on}
                                                    suffix={true}
                                                />
                                            );
                                        }
                                    },
                                    {
                                        label: 'Expires',
                                        tooltip: 'Valid Until',
                                        sortType: 'date',
                                        align: 'left',
                                        width: '150px',
                                        virtual: true,
                                        virtualKey: 'valid_until',
                                        sortable: true,
                                        renderer: (obj) => {
                                            return (
                                                <DataTableRelativeDateField
                                                    date={obj.valid_until}
                                                />
                                            );
                                        }
                                    },
                                    {
                                        label: 'Duration',
                                        tooltip: 'Total Ban Duration',
                                        sortType: 'number',
                                        align: 'left',
                                        width: '150px',
                                        virtual: true,
                                        virtualKey: 'duration',
                                        renderer: (row) => {
                                            const dur = intervalToDuration({
                                                start: row.created_on,
                                                end: row.valid_until
                                            });
                                            const durationText =
                                                dur.years && dur.years > 5
                                                    ? 'Permanent'
                                                    : formatDuration(dur);
                                            return (
                                                <Typography
                                                    variant={'body1'}
                                                    overflow={'hidden'}
                                                >
                                                    {durationText}
                                                </Typography>
                                            );
                                        }
                                    },
                                    {
                                        label: 'Act.',
                                        tooltip: 'Actions',
                                        sortKey: 'reason',
                                        sortable: false,
                                        align: 'left',
                                        renderer: (row) => (
                                            <ButtonGroup fullWidth>
                                                <IconButton
                                                    color={'warning'}
                                                    onClick={async () => {
                                                        await onEditCIDR(row);
                                                    }}
                                                >
                                                    <Tooltip
                                                        title={'Edit CIDR Ban'}
                                                    >
                                                        <EditIcon />
                                                    </Tooltip>
                                                </IconButton>
                                                <IconButton
                                                    color={'success'}
                                                    onClick={async () => {
                                                        await onUnbanCIDR(
                                                            row.net_id
                                                        );
                                                    }}
                                                >
                                                    <Tooltip
                                                        title={
                                                            'Remove CIDR Ban'
                                                        }
                                                    >
                                                        <UndoIcon />
                                                    </Tooltip>
                                                </IconButton>
                                            </ButtonGroup>
                                        )
                                    }
                                ]}
                                defaultSortColumn={'net_id'}
                                rowsPerPage={RowsPerPage.TwentyFive}
                                rows={banCIDRs}
                            />
                        </TabPanel>
                        <TabPanel value={value} index={2}>
                            <DataTable<IAPIBanASNRecord>
                                columns={[
                                    {
                                        label: '#',
                                        tooltip: 'Ban ID',
                                        sortKey: 'ban_asn_id',
                                        sortable: true,
                                        align: 'left',
                                        queryValue: (o) => `${o.ban_asn_id}`,
                                        renderer: (obj) => (
                                            <Typography variant={'body1'}>
                                                #{obj.ban_asn_id.toString()}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'ASN',
                                        tooltip: 'Autonomous System Numbers',
                                        sortKey: 'as_num',
                                        sortable: true,
                                        align: 'left',
                                        queryValue: (o) => `${o.as_num}`,
                                        renderer: (row) => (
                                            <Typography variant={'body1'}>
                                                {row.as_num}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'Reason',
                                        tooltip: 'Reason',
                                        sortKey: 'reason',
                                        sortable: true,
                                        align: 'left',
                                        queryValue: (o) => BanReason[o.reason],
                                        renderer: (row) => (
                                            <Typography variant={'body1'}>
                                                {BanReason[row.reason]}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'Custom Reason',
                                        tooltip: 'Custom',
                                        sortKey: 'reason_text',
                                        sortable: false,
                                        align: 'left',
                                        queryValue: (o) => o.reason_text
                                    },
                                    {
                                        label: 'Created',
                                        tooltip: 'Created On',
                                        sortType: 'date',
                                        align: 'left',
                                        width: '150px',
                                        virtual: true,
                                        virtualKey: 'created_on',
                                        renderer: (obj) => {
                                            return (
                                                <Typography variant={'body1'}>
                                                    {format(
                                                        obj.created_on,
                                                        'yyyy-MM-dd'
                                                    )}
                                                </Typography>
                                            );
                                        }
                                    },
                                    {
                                        label: 'Expires',
                                        tooltip: 'Valid Until',
                                        sortType: 'date',
                                        align: 'left',
                                        width: '150px',
                                        virtual: true,
                                        virtualKey: 'valid_until',
                                        sortable: true,
                                        renderer: (obj) => {
                                            return (
                                                <Typography variant={'body1'}>
                                                    {format(
                                                        obj.valid_until,
                                                        'yyyy-MM-dd'
                                                    )}
                                                </Typography>
                                            );
                                        }
                                    },
                                    {
                                        label: 'Duration',
                                        tooltip: 'Total Ban Duration',
                                        sortType: 'number',
                                        align: 'left',
                                        width: '150px',
                                        virtual: true,
                                        virtualKey: 'duration',
                                        renderer: (row) => {
                                            const dur = intervalToDuration({
                                                start: row.created_on,
                                                end: row.valid_until
                                            });
                                            const durationText =
                                                dur.years && dur.years > 5
                                                    ? 'Permanent'
                                                    : formatDuration(dur);
                                            return (
                                                <Typography
                                                    variant={'body1'}
                                                    overflow={'hidden'}
                                                >
                                                    {durationText}
                                                </Typography>
                                            );
                                        }
                                    },
                                    {
                                        label: 'Act.',
                                        tooltip: 'Actions',
                                        sortKey: 'reason',
                                        sortable: false,
                                        align: 'left',
                                        renderer: (row) => (
                                            <ButtonGroup fullWidth>
                                                <IconButton
                                                    color={'warning'}
                                                    onClick={async () =>
                                                        await onEditASN(row)
                                                    }
                                                >
                                                    <Tooltip
                                                        title={'Edit ASN Ban'}
                                                    >
                                                        <EditIcon />
                                                    </Tooltip>
                                                </IconButton>
                                                <IconButton
                                                    color={'success'}
                                                    onClick={async () =>
                                                        await onUnbanASN(
                                                            row.as_num
                                                        )
                                                    }
                                                >
                                                    <Tooltip
                                                        title={
                                                            'Remove CIDR Ban'
                                                        }
                                                    >
                                                        <UndoIcon />
                                                    </Tooltip>
                                                </IconButton>
                                            </ButtonGroup>
                                        )
                                    }
                                ]}
                                defaultSortColumn={'ban_asn_id'}
                                rowsPerPage={RowsPerPage.TwentyFive}
                                rows={banASNs}
                            />
                        </TabPanel>

                        <TabPanel value={value} index={3}>
                            <DataTable<IAPIBanGroupRecord>
                                columns={[
                                    {
                                        label: '#',
                                        tooltip: 'Ban ID',
                                        sortKey: 'ban_group_id',
                                        sortable: true,
                                        align: 'left',
                                        queryValue: (o) => `${o.ban_group_id}`,
                                        renderer: (obj) => (
                                            <Typography variant={'body1'}>
                                                #{obj.ban_group_id.toString()}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'SteamID',
                                        tooltip:
                                            'SteamID of the primary target',
                                        sortKey: 'target_id',
                                        sortable: true,
                                        align: 'left',
                                        queryValue: (o) => `${o.target_id}`,
                                        renderer: (row) => (
                                            <Typography variant={'body1'}>
                                                {row.target_id}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'GroupID',
                                        tooltip: 'GroupID',
                                        sortKey: 'group_id',
                                        sortable: true,
                                        align: 'left',
                                        queryValue: (o) => `${o.target_id}`,
                                        renderer: (row) => (
                                            <Typography variant={'body1'}>
                                                {row.group_id}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'Note',
                                        tooltip: 'Mod Note',
                                        sortKey: 'note',
                                        sortable: false,
                                        align: 'left',
                                        queryValue: (row) => row.note,
                                        renderer: (row) => (
                                            <Typography variant={'body1'}>
                                                {row.note}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'Created',
                                        tooltip: 'Created On',
                                        sortType: 'date',
                                        align: 'left',
                                        width: '150px',
                                        virtual: true,
                                        virtualKey: 'created_on',
                                        renderer: (obj) => {
                                            return (
                                                <Typography variant={'body1'}>
                                                    {format(
                                                        obj.created_on,
                                                        'yyyy-MM-dd'
                                                    )}
                                                </Typography>
                                            );
                                        }
                                    },
                                    {
                                        label: 'Expires',
                                        tooltip: 'Valid Until',
                                        sortType: 'date',
                                        align: 'left',
                                        width: '150px',
                                        virtual: true,
                                        virtualKey: 'valid_until',
                                        sortable: true,
                                        renderer: (obj) => {
                                            return (
                                                <Typography variant={'body1'}>
                                                    {format(
                                                        obj.valid_until,
                                                        'yyyy-MM-dd'
                                                    )}
                                                </Typography>
                                            );
                                        }
                                    },
                                    {
                                        label: 'Duration',
                                        tooltip: 'Total Ban Duration',
                                        sortType: 'number',
                                        align: 'left',
                                        width: '150px',
                                        virtual: true,
                                        virtualKey: 'duration',
                                        renderer: (row) => {
                                            const dur = intervalToDuration({
                                                start: row.created_on,
                                                end: row.valid_until
                                            });
                                            const durationText =
                                                dur.years && dur.years > 5
                                                    ? 'Permanent'
                                                    : formatDuration(dur);
                                            return (
                                                <Typography
                                                    variant={'body1'}
                                                    overflow={'hidden'}
                                                >
                                                    {durationText}
                                                </Typography>
                                            );
                                        }
                                    },
                                    {
                                        label: 'Act.',
                                        tooltip: 'Actions',
                                        sortKey: 'reason',
                                        sortable: false,
                                        align: 'left',
                                        renderer: (row) => (
                                            <ButtonGroup fullWidth>
                                                <IconButton
                                                    color={'warning'}
                                                    onClick={async () => {
                                                        await onEditGroup(row);
                                                    }}
                                                >
                                                    <Tooltip title={'Edit Ban'}>
                                                        <EditIcon />
                                                    </Tooltip>
                                                </IconButton>
                                                <IconButton
                                                    color={'success'}
                                                    onClick={async () => {
                                                        await onUnbanGroup(
                                                            row.ban_group_id
                                                        );
                                                    }}
                                                >
                                                    <Tooltip
                                                        title={'Remove Ban'}
                                                    >
                                                        <UndoIcon />
                                                    </Tooltip>
                                                </IconButton>
                                            </ButtonGroup>
                                        )
                                    }
                                ]}
                                defaultSortColumn={'ban_group_id'}
                                rowsPerPage={RowsPerPage.TwentyFive}
                                rows={banGroups}
                            />
                        </TabPanel>
                    </Paper>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};
