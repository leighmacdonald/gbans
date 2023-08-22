import React, { useCallback, useEffect, useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import ButtonGroup from '@mui/material/ButtonGroup';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import UndoIcon from '@mui/icons-material/Undo';
import EditIcon from '@mui/icons-material/Edit';
import Tooltip from '@mui/material/Tooltip';
import GavelIcon from '@mui/icons-material/Gavel';
import Box from '@mui/material/Box';
import VisibilityIcon from '@mui/icons-material/Visibility';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import { useTheme } from '@mui/material/styles';
import { useNavigate } from 'react-router-dom';
import format from 'date-fns/format';
import { formatDuration, intervalToDuration } from 'date-fns';
import { TabPanel } from '../component/TabPanel';
import { DataTable, RowsPerPage } from '../component/DataTable';
import { PersonCell } from '../component/PersonCell';
import { BanCIDRModal } from '../component/BanCIDRModal';
import { BanASNModal } from '../component/BanASNModal';
import { BanGroupModal } from '../component/BanGroupModal';
import { UnbanCIDRModal } from '../component/UnbanCIDRModal';
import { UnbanASNModal } from '../component/UnbanASNModal';
import { UnbanGroupModal } from '../component/UnbanGroupModal';
import { UnbanSteamModal } from '../component/UnbanSteamModal';
import { BanSteamModal } from '../component/BanSteamModal';
import { steamIdQueryValue } from '../util/text';
import {
    apiGetBansSteam,
    apiGetBansASN,
    apiGetBansCIDR,
    apiGetBansGroups,
    BanReason,
    IAPIBanASNRecord,
    IAPIBanCIDRRecord,
    IAPIBanGroupRecord,
    IAPIBanRecordProfile
} from '../api';
import {
    DataTableRelativeDateField,
    isPermanentBan
} from '../component/DataTableRelativeDateField';

export const AdminBan = () => {
    const theme = useTheme();
    const [bans, setBans] = useState<IAPIBanRecordProfile[]>([]);
    const [banGroups, setBanGroups] = useState<IAPIBanGroupRecord[]>([]);
    const [banCIDRs, setBanCIDRs] = useState<IAPIBanCIDRRecord[]>([]);
    const [banASNs, setBanASNs] = useState<IAPIBanASNRecord[]>([]);
    const [currentBan, setCurrentBan] = useState<IAPIBanRecordProfile>();
    const [currentBanCIDR, setCurrentBanCIDR] = useState<IAPIBanCIDRRecord>();
    const [currentBanASN, setCurrentBanASN] = useState<IAPIBanASNRecord>();
    const [currentBanGroup, setCurrentBanGroup] =
        useState<IAPIBanGroupRecord>();
    const [banSteamModalOpen, setBanSteamModalOpen] = useState(false);
    const [banCIDRModalOpen, setBanCIDRModalOpen] = useState(false);
    const [banASNModalOpen, setBanASNModalOpen] = useState(false);
    const [banGroupModalOpen, setBanGroupModalOpen] = useState(false);
    const [unbanSteamModalOpen, setUnbanSteamModalOpen] = useState(false);
    const [unbanCIDRModalOpen, setUnbanCIDRModalOpen] = useState(false);
    const [unbanASNModalOpen, setUnbanASNModalOpen] = useState(false);
    const [unbanGroupModalOpen, setUnbanGroupModalOpen] = useState(false);
    const [value, setValue] = React.useState<number>(0);
    const navigate = useNavigate();

    const loadBansGroup = useCallback(() => {
        apiGetBansGroups({ desc: true, order_by: 'ban_group_id' }).then(
            (newGroupBans) => {
                setBanGroups(newGroupBans.result || []);
            }
        );
    }, []);

    const loadBansCIDR = useCallback(() => {
        apiGetBansCIDR({ desc: true, order_by: 'net_id' }).then(
            (newBansCIDR) => {
                setBanCIDRs(newBansCIDR.result || []);
            }
        );
    }, []);

    const loadBansASN = useCallback(() => {
        apiGetBansASN({ desc: true, order_by: 'ban_asn_id' }).then(
            (newBansASN) => {
                setBanASNs(newBansASN.result || []);
            }
        );
    }, []);

    const loadBansSteam = useCallback(() => {
        apiGetBansSteam({ desc: true, order_by: 'ban_id' }).then((newBans) => {
            setBans(newBans || []);
        });
    }, []);

    useEffect(() => {
        loadBansSteam();
        loadBansCIDR();
        loadBansASN();
        loadBansGroup();
    }, [loadBansASN, loadBansCIDR, loadBansGroup, loadBansSteam]);

    return (
        <Box>
            <BanSteamModal
                open={banSteamModalOpen}
                setOpen={setBanSteamModalOpen}
            />
            <BanCIDRModal
                open={banCIDRModalOpen}
                setOpen={setBanCIDRModalOpen}
            />
            <BanASNModal open={banASNModalOpen} setOpen={setBanASNModalOpen} />
            <BanGroupModal
                open={banGroupModalOpen}
                setOpen={setBanGroupModalOpen}
            />
            {currentBan && (
                <UnbanSteamModal
                    banId={currentBan.ban_id}
                    personaName={currentBan.personaname}
                    open={unbanSteamModalOpen}
                    setOpen={setUnbanSteamModalOpen}
                    onSuccess={() => {
                        setUnbanSteamModalOpen(false);
                        setBans((bans) => {
                            return bans.filter(
                                (b) => b.ban_id != currentBan?.ban_id
                            );
                        });
                    }}
                />
            )}
            {currentBanCIDR && (
                <UnbanCIDRModal
                    record={currentBanCIDR}
                    open={unbanCIDRModalOpen}
                    setOpen={setUnbanCIDRModalOpen}
                    onSuccess={() => {
                        setUnbanCIDRModalOpen(false);
                        setBanCIDRs((bans) => {
                            return bans.filter(
                                (b) => b.net_id != currentBanCIDR?.net_id
                            );
                        });
                    }}
                />
            )}
            {currentBanASN && (
                <UnbanASNModal
                    record={currentBanASN}
                    open={unbanASNModalOpen}
                    setOpen={setUnbanASNModalOpen}
                    onSuccess={() => {
                        setUnbanASNModalOpen(false);
                        setBanASNs((bans) => {
                            return bans.filter(
                                (b) => b.ban_asn_id != currentBanASN?.ban_asn_id
                            );
                        });
                    }}
                />
            )}
            {currentBanGroup && (
                <UnbanGroupModal
                    record={currentBanGroup}
                    open={unbanGroupModalOpen}
                    setOpen={setUnbanGroupModalOpen}
                    onSuccess={() => {
                        setUnbanGroupModalOpen(false);
                        setBanGroups((bans) => {
                            return bans.filter(
                                (b) =>
                                    b.ban_group_id !=
                                    currentBanGroup?.ban_group_id
                            );
                        });
                    }}
                />
            )}
            <ButtonGroup>
                <Button
                    variant={'contained'}
                    color={'secondary'}
                    startIcon={<GavelIcon />}
                    sx={{ marginRight: 2 }}
                    onClick={() => {
                        setBanSteamModalOpen(true);
                    }}
                >
                    Steam
                </Button>
                <Button
                    variant={'contained'}
                    color={'secondary'}
                    startIcon={<GavelIcon />}
                    sx={{ marginRight: 2 }}
                    onClick={() => {
                        setBanCIDRModalOpen(true);
                    }}
                >
                    CIDR
                </Button>
                <Button
                    variant={'contained'}
                    color={'secondary'}
                    startIcon={<GavelIcon />}
                    sx={{ marginRight: 2 }}
                    onClick={() => {
                        setBanASNModalOpen(true);
                    }}
                >
                    ASN
                </Button>
                <Button
                    variant={'contained'}
                    color={'secondary'}
                    startIcon={<GavelIcon />}
                    sx={{ marginRight: 2 }}
                    onClick={() => {
                        setBanGroupModalOpen(true);
                    }}
                >
                    Group
                </Button>
            </ButtonGroup>
            <Paper>
                <Box
                    marginTop={2}
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
                                        queryValue: (o) => `${o.ban_id}`,
                                        renderer: (obj) => (
                                            <Typography variant={'body1'}>
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
                                            steamIdQueryValue(o.target_id),
                                        renderer: (row) => (
                                            <PersonCell
                                                steam_id={row.target_id}
                                                personaname={row.personaname}
                                                avatar={row.avatar}
                                            />
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
                                            return isPermanentBan(
                                                row.created_on,
                                                row.valid_until
                                            ) ? (
                                                'Permanent'
                                            ) : (
                                                <DataTableRelativeDateField
                                                    date={row.created_on}
                                                    compareDate={
                                                        row.valid_until
                                                    }
                                                />
                                            );
                                        }
                                    },
                                    {
                                        label: 'Rep.',
                                        tooltip: 'Report',
                                        sortable: false,
                                        align: 'left',
                                        width: '20px',
                                        queryValue: (o) => `${o.report_id}`,
                                        renderer: (row) =>
                                            row.report_id > 0 ? (
                                                <Tooltip title={'View Report'}>
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
                                                    <Tooltip title={'View'}>
                                                        <VisibilityIcon />
                                                    </Tooltip>
                                                </IconButton>
                                                <IconButton color={'warning'}>
                                                    <Tooltip title={'Edit Ban'}>
                                                        <EditIcon />
                                                    </Tooltip>
                                                </IconButton>
                                                <IconButton
                                                    color={'success'}
                                                    onClick={() => {
                                                        setCurrentBan(row);
                                                        setUnbanSteamModalOpen(
                                                            true
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
                                            <Typography variant={'body1'}>
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
                                        <IconButton color={'warning'}>
                                            <Tooltip title={'Edit CIDR Ban'}>
                                                <EditIcon />
                                            </Tooltip>
                                        </IconButton>
                                        <IconButton
                                            color={'success'}
                                            onClick={() => {
                                                setCurrentBanCIDR(row);
                                                setUnbanCIDRModalOpen(true);
                                            }}
                                        >
                                            <Tooltip title={'Remove CIDR Ban'}>
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
                                        <IconButton color={'warning'}>
                                            <Tooltip title={'Edit CIDR Ban'}>
                                                <EditIcon />
                                            </Tooltip>
                                        </IconButton>
                                        <IconButton
                                            color={'success'}
                                            onClick={() => {
                                                setCurrentBanASN(row);
                                                setUnbanASNModalOpen(true);
                                            }}
                                        >
                                            <Tooltip title={'Remove CIDR Ban'}>
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
                                label: 'GroupID',
                                tooltip: 'GroupID',
                                sortKey: 'target_id',
                                sortable: true,
                                align: 'left',
                                queryValue: (o) => `${o.target_id}`,
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {row.target_id.toString()}
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
                                        <IconButton color={'warning'}>
                                            <Tooltip title={'Edit Ban'}>
                                                <EditIcon />
                                            </Tooltip>
                                        </IconButton>
                                        <IconButton
                                            color={'success'}
                                            onClick={() => {
                                                setCurrentBanGroup(row);
                                                setUnbanGroupModalOpen(true);
                                            }}
                                        >
                                            <Tooltip title={'Remove Ban'}>
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
        </Box>
    );
};
