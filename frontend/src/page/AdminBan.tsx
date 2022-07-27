import React, { useCallback, useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import { apiGetBans, BanReason, IAPIBanRecord } from '../api';
import { DataTable } from '../component/DataTable';
import { PersonCell } from '../component/PersonCell';
import format from 'date-fns/format';
import { formatDuration, intervalToDuration, parseISO } from 'date-fns';
import ButtonGroup from '@mui/material/ButtonGroup';
import Button from '@mui/material/Button';
import { useNavigate } from 'react-router-dom';
import IconButton from '@mui/material/IconButton';
import UndoIcon from '@mui/icons-material/Undo';
import EditIcon from '@mui/icons-material/Edit';
import Tooltip from '@mui/material/Tooltip';
import GavelIcon from '@mui/icons-material/Gavel';
import { BanModal } from '../component/BanModal';
import Box from '@mui/material/Box';
import { UnbanModal } from '../component/UnbanModal';

export const AdminBan = (): JSX.Element => {
    const [bans, setBans] = useState<IAPIBanRecord[]>([]);
    const [currentBan, setCurrentBan] = useState<IAPIBanRecord>();
    const [modalOpen, setModalOpen] = useState(false);
    const [unbanModalOpen, setUnbanModalOpen] = useState(false);
    const navigate = useNavigate();

    const loadBans = useCallback(() => {
        apiGetBans({ desc: true, order_by: 'ban_id' }).then((newBans) => {
            setBans(newBans || []);
        });
    }, []);

    useEffect(() => {
        loadBans();
    }, [loadBans]);

    return (
        <Box marginTop={3}>
            <BanModal
                open={modalOpen}
                setOpen={setModalOpen}
                onSuccess={() => {
                    loadBans();
                    setModalOpen(false);
                }}
            />
            {currentBan && (
                <UnbanModal
                    banRecord={currentBan}
                    open={unbanModalOpen}
                    setOpen={setUnbanModalOpen}
                    onSuccess={() => {
                        setUnbanModalOpen(false);
                        setBans((bans) => {
                            return bans.filter(
                                (b) => b.ban_id != currentBan?.ban_id
                            );
                        });
                    }}
                />
            )}
            <ButtonGroup>
                <Button
                    variant={'contained'}
                    color={'error'}
                    startIcon={<GavelIcon />}
                    onClick={() => {
                        setModalOpen(true);
                    }}
                >
                    Ban Player
                </Button>
            </ButtonGroup>
            <Grid container spacing={3} paddingTop={3}>
                <Grid item xs={12}>
                    <Paper elevation={1}>
                        <DataTable
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
                                    queryValue: (o) => `${o.personaname}`,
                                    renderer: (row) => (
                                        <PersonCell
                                            steam_id={row.steam_id}
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
                                                    parseISO(
                                                        obj.created_on as any as string
                                                    ),
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
                                                    parseISO(
                                                        obj.valid_until as any as string
                                                    ),
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
                                        const t0 = parseISO(
                                            row.created_on as any as string
                                        );
                                        const t1 = parseISO(
                                            row.valid_until as any as string
                                        );
                                        const dur = intervalToDuration({
                                            start: t0,
                                            end: t1
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
                                            <IconButton color={'warning'}>
                                                <Tooltip title={'Edit Ban'}>
                                                    <EditIcon />
                                                </Tooltip>
                                            </IconButton>
                                            <IconButton
                                                color={'success'}
                                                onClick={() => {
                                                    setCurrentBan(row);
                                                    setUnbanModalOpen(true);
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
                            defaultSortColumn={'ban_id'}
                            rowsPerPage={100}
                            rows={bans}
                        />
                    </Paper>
                </Grid>
            </Grid>
        </Box>
    );
};
