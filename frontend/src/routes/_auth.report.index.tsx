import { JSX, useMemo, useState } from 'react';
import EditNotificationsIcon from '@mui/icons-material/EditNotifications';
import HistoryIcon from '@mui/icons-material/History';
import InfoIcon from '@mui/icons-material/Info';
import VisibilityIcon from '@mui/icons-material/Visibility';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import IconButton from '@mui/material/IconButton';
import InputLabel from '@mui/material/InputLabel';
import Link from '@mui/material/Link';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation, useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate, useRouteContext } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import {
    apiCreateReport,
    apiGetReports,
    BanReason,
    BanReasons,
    banReasonsCollection,
    CreateReportRequest,
    PlayerProfile,
    ReportStatus,
    reportStatusString,
    ReportWithAuthor
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { PlayerMessageContext } from '../component/PlayerMessageContext.tsx';
import { ReportStatusIcon } from '../component/ReportStatusIcon.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { MarkdownField } from '../component/field/MarkdownField.tsx';
import { SteamIDField } from '../component/field/SteamIDField.tsx';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';
import { makeSteamidValidators } from '../util/validator/makeSteamidValidators.ts';

const reportSchema = z.object({
    ...commonTableSearchSchema,
    rows: z.number().optional(),
    sortColumn: z.enum(['report_status', 'created_on']).optional(),
    report_status: z.nativeEnum(ReportStatus).optional(),
    steam_id: z.string().optional(),
    demo_name: z.string().optional(),
    person_message_id: z.number().optional()
});

export const Route = createFileRoute('/_auth/report/')({
    component: ReportCreate,
    validateSearch: (search) => reportSchema.parse(search)
});

function ReportCreate() {
    const defaultRows = RowsPerPage.Ten;
    const { profile, userSteamID } = useRouteContext({ from: '/_auth/report/' });
    const { page, sortColumn, report_status, rows, sortOrder } = Route.useSearch();

    const canReport = useMemo(() => {
        const user = profile();
        return user.steam_id && user.ban_id == 0;
    }, [profile]);

    const { data: logs, isLoading } = useQuery({
        queryKey: ['history', { page, userSteamID }],
        queryFn: async () => {
            return await apiGetReports({
                source_id: userSteamID,
                limit: rows ?? defaultRows,
                offset: (page ?? 0) * (rows ?? defaultRows),
                order_by: sortColumn ?? 'created_on',
                desc: (sortOrder ?? 'desc') == 'desc',
                report_status: report_status ?? ReportStatus.Any
            });
        }
    });

    return (
        <Grid container spacing={3}>
            <Grid xs={12} md={8}>
                <Stack spacing={2}>
                    {canReport && <ReportCreateForm />}
                    {!canReport && (
                        <ContainerWithHeader title={'Permission Denied'}>
                            <Typography variant={'body1'} padding={2}>
                                You are unable to report players while you are currently banned/muted.
                            </Typography>
                            <ButtonGroup sx={{ padding: 2 }}>
                                <Button
                                    component={RouterLink}
                                    variant={'contained'}
                                    color={'primary'}
                                    to={`/ban/${profile().ban_id}`}
                                >
                                    Appeal Ban
                                </Button>
                            </ButtonGroup>
                        </ContainerWithHeader>
                    )}
                    <ContainerWithHeader title={'Your Report History'} iconLeft={<HistoryIcon />}>
                        {isLoading ? (
                            <LoadingPlaceholder />
                        ) : (
                            <UserReportHistory history={logs ?? { data: [], count: 0 }} isLoading={isLoading} />
                        )}
                        <Paginator page={page ?? 0} rows={rows ?? defaultRows} data={logs} path={'/report'} />
                    </ContainerWithHeader>
                </Stack>
            </Grid>
            <Grid xs={12} md={4}>
                <ContainerWithHeader title={'Reporting Guide'} iconLeft={<InfoIcon />}>
                    <List>
                        <ListItem>
                            <ListItemText>
                                Once your report is posted, it will be reviewed by a moderator. If further details are
                                required you will be notified about it.
                            </ListItemText>
                        </ListItem>
                        <ListItem>
                            <ListItemText>
                                If you wish to link to a specific SourceTV recording, you can find them listed{' '}
                                <Link component={RouterLink} to={'/stv'}>
                                    here
                                </Link>
                                . Once you find the recording you want, you may select the report icon which will open a
                                new report with the demo attached. From there you will optionally be able to enter a
                                specific tick if you have one.
                            </ListItemText>
                        </ListItem>
                        <ListItem>
                            <ListItemText>
                                Reports that are made in bad faith, or otherwise are considered to be trolling will be
                                closed, and the reporter will be banned.
                            </ListItemText>
                        </ListItem>

                        <ListItem>
                            <ListItemText>
                                Its only possible to open a single report against a particular player. If you wish to
                                add more evidence or discuss further an existing report, please open the existing report
                                and add it by creating a new message there. You can see your current report history
                                below.
                            </ListItemText>
                        </ListItem>
                    </List>
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<ReportWithAuthor>();

const UserReportHistory = ({ history, isLoading }: { history: LazyResult<ReportWithAuthor>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('report_status', {
            header: () => <TableHeadingCell name={'Status'} />,
            cell: (info) => {
                return (
                    <Stack direction={'row'} spacing={1}>
                        <ReportStatusIcon reportStatus={info.getValue()} />
                        <Typography variant={'body1'}>{reportStatusString(info.getValue())}</Typography>
                    </Stack>
                );
            },
            footer: () => <TableHeadingCell name={'Server'} />
        }),
        columnHelper.accessor('subject', {
            header: () => <TableHeadingCell name={'Player'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={history.data[info.row.index].subject.steam_id}
                    personaname={history.data[info.row.index].subject.personaname}
                    avatar_hash={history.data[info.row.index].subject.avatarhash}
                />
            ),
            footer: () => <TableHeadingCell name={'Created'} />
        }),
        columnHelper.accessor('report_id', {
            header: () => <TableHeadingCell name={'View'} />,
            cell: (info) => (
                <ButtonGroup>
                    <IconButton
                        color={'primary'}
                        component={RouterLink}
                        to={`/report/$reportId`}
                        params={{ reportId: info.getValue() }}
                    >
                        <Tooltip title={'View'}>
                            <VisibilityIcon />
                        </Tooltip>
                    </IconButton>
                </ButtonGroup>
            ),
            footer: () => <TableHeadingCell name={'Name'} />
        })
    ];

    const table = useReactTable({
        data: history.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};

const validationSchema = z.object({
    steam_id: z.string(),
    body_md: z.string().min(10, 'Message too short (min 10)'),
    reason: z.nativeEnum(BanReason),
    reason_text: z.string().optional(),

    //person_message_id: yup.number().min(1, 'Invalid message id').optional()
    demo_name: z.string().optional(),
    demo_tick: z.number().min(0, 'invalid demo tick value').optional()
});

export const ReportCreateForm = (): JSX.Element => {
    const { demo_name, steam_id, person_message_id } = Route.useSearch();
    const [validatedProfile, setValidatedProfile] = useState<PlayerProfile>();

    const mutation = useMutation({
        mutationFn: async (variables: CreateReportRequest) => {
            return await apiCreateReport(variables);
        },
        onSuccess: async (data) => {
            await navigate({ to: '/report/$reportId', params: { reportId: String(data.report_id) } });
        }
    });

    const navigate = useNavigate();

    const form = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({
                demo_name: value.demo_name,
                target_id: validatedProfile?.player.steam_id ?? '',
                demo_tick: value.demo_tick,
                reason: value.reason,
                reason_text: value.reason_text,
                description: value.body_md,
                person_message_id: value.person_message_id
            });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: validationSchema
        },
        defaultValues: {
            body_md: '',
            demo_name: demo_name ?? '',
            demo_tick: 0,
            person_message_id: person_message_id ?? 0,
            steam_id: steam_id ?? '',
            reason: BanReason.Custom,
            reason_text: ''
        }
    });

    // const onSubmit = useCallback(
    //     async (values: ReportValues, formikHelpers: FormikHelpers<ReportValues>) => {
    //         try {
    //             const report = await apiCreateReport({
    //                 demo_name: values.demo_name,
    //                 demo_tick: values.demo_tick ?? 0,
    //                 description: values.body_md,
    //                 reason_text: values.reason_text,
    //                 target_id: values.steam_id,
    //                 person_message_id: values.person_message_id,
    //                 reason: values.reason
    //             });
    //             await navigate({ to: `/report/${report.report_id}` });
    //             formikHelpers.resetForm();
    //         } catch (e) {
    //             logErr(e);
    //         }
    //     },
    //     [navigate]
    // );

    return (
        <ContainerWithHeader
            title={'Create a New Report'}
            iconLeft={<EditNotificationsIcon />}
            spacing={2}
            marginTop={3}
        >
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid xs={12}>
                        <form.Field
                            name={'steam_id'}
                            validators={makeSteamidValidators(setValidatedProfile)}
                            children={({ state, handleChange, handleBlur }) => {
                                return (
                                    <SteamIDField
                                        state={state}
                                        handleBlur={handleBlur}
                                        handleChange={handleChange}
                                        fullwidth={true}
                                        label={'SteamID'}
                                    />
                                );
                            }}
                        />
                    </Grid>
                    <Grid xs={6}>
                        <form.Field
                            name={'reason'}
                            validators={{
                                onChange: z.nativeEnum(BanReason, { message: 'Invalid ban reason' })
                            }}
                            children={({ state, handleChange, handleBlur }) => {
                                return (
                                    <>
                                        <FormControl fullWidth>
                                            <InputLabel id="server-select-label">Reason</InputLabel>
                                            <Select
                                                fullWidth
                                                value={state.value}
                                                label="Servers"
                                                onChange={(e) => {
                                                    handleChange(Number(e.target.value));
                                                }}
                                                onBlur={handleBlur}
                                                error={state.meta.touchedErrors.length > 0}
                                            >
                                                {banReasonsCollection.map((r) => (
                                                    <MenuItem value={r} key={`reason-${r}`}>
                                                        {BanReasons[r]}
                                                    </MenuItem>
                                                ))}
                                            </Select>
                                            {state.meta.touchedErrors.length > 0 && (
                                                <FormHelperText>Error</FormHelperText>
                                            )}
                                        </FormControl>
                                    </>
                                );
                            }}
                        />
                    </Grid>

                    <Grid xs={6}>
                        <form.Field
                            name={'reason_text'}
                            children={({ state, handleChange, handleBlur }) => {
                                return (
                                    <>
                                        <TextField
                                            fullWidth
                                            label="Custom Reason"
                                            value={state.value}
                                            onChange={(e) => handleChange(e.target.value)}
                                            onBlur={handleBlur}
                                            variant="outlined"
                                        />
                                    </>
                                );
                            }}
                        />
                    </Grid>
                    <Grid md={6}>
                        <form.Field
                            name={'demo_name'}
                            children={({ state, handleChange, handleBlur }) => {
                                return (
                                    <TextField
                                        fullWidth
                                        label="Demo Name"
                                        value={state.value}
                                        onChange={(e) => handleChange(e.target.value)}
                                        onBlur={handleBlur}
                                        variant="outlined"
                                    />
                                );
                            }}
                        />
                    </Grid>
                    <Grid md={6}>
                        <form.Field
                            name={'demo_tick'}
                            children={({ state, handleChange, handleBlur }) => {
                                return (
                                    <TextField
                                        fullWidth
                                        label="Demo Tick"
                                        value={state.value}
                                        onChange={(e) => handleChange(Number(e.target.value))}
                                        onBlur={handleBlur}
                                        variant="outlined"
                                    />
                                );
                            }}
                        />
                    </Grid>

                    {person_message_id != undefined && person_message_id > 0 && (
                        <Grid md={12}>
                            <PlayerMessageContext playerMessageId={person_message_id} padding={5} />
                        </Grid>
                    )}
                    <Grid md={12}>
                        <Box minHeight={365}>
                            <form.Field
                                name={'body_md'}
                                children={(props) => {
                                    return <MarkdownField {...props} label={'Message (Markdown)'} />;
                                }}
                            />
                        </Box>
                    </Grid>
                    <Grid xs={12}>
                        <form.Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => (
                                <Buttons canSubmit={canSubmit} isSubmitting={isSubmitting} reset={form.reset} />
                            )}
                        />
                    </Grid>
                </Grid>
            </form>
        </ContainerWithHeader>
    );
};
