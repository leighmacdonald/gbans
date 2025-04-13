import { JSX, useMemo, useState } from 'react';
import EditNotificationsIcon from '@mui/icons-material/EditNotifications';
import HistoryIcon from '@mui/icons-material/History';
import InfoIcon from '@mui/icons-material/Info';
import VisibilityIcon from '@mui/icons-material/Visibility';
import ButtonGroup from '@mui/material/ButtonGroup';
import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import Grid from '@mui/material/Grid';
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
import { useForm } from '@tanstack/react-form';
import { useMutation, useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate, useRouteContext } from '@tanstack/react-router';
import {
    createColumnHelper,
    getCoreRowModel,
    getPaginationRowModel,
    getSortedRowModel,
    SortingState,
    useReactTable
} from '@tanstack/react-table';
import { z } from 'zod';
import {
    apiCreateReport,
    apiGetUserReports,
    BanReason,
    BanReasons,
    banReasonsCollection,
    CreateReportRequest,
    PlayerProfile,
    reportStatusString,
    ReportWithAuthor
} from '../api';
import { ButtonLink } from '../component/ButtonLink.tsx';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { IconButtonLink } from '../component/IconButtonLink.tsx';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { PlayerMessageContext } from '../component/PlayerMessageContext.tsx';
import { ReportStatusIcon } from '../component/ReportStatusIcon.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { MarkdownField, mdEditorRef } from '../component/field/MarkdownField.tsx';
import { SteamIDField } from '../component/field/SteamIDField.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { commonTableSearchSchema, initPagination, RowsPerPage } from '../util/table.ts';

const reportSchema = z.object({
    ...commonTableSearchSchema,
    rows: z.number().optional(),
    sortColumn: z.enum(['report_status', 'created_on']).optional(),
    steam_id: z.string().optional(),
    demo_id: z.number({ coerce: true }).optional(),
    person_message_id: z.number().optional()
});

export const Route = createFileRoute('/_auth/report/')({
    component: ReportCreate,
    validateSearch: (search) => reportSchema.parse(search)
});

function ReportCreate() {
    const { profile } = useRouteContext({ from: '/_auth/report/' });

    const canReport = useMemo(() => {
        return profile.steam_id && profile.ban_id == 0;
    }, [profile]);

    const { data: logs, isLoading } = useQuery({
        queryKey: ['history', { steam_id: profile.steam_id }],
        queryFn: async () => {
            return await apiGetUserReports();
        }
    });

    return (
        <Grid container spacing={2}>
            <Title>Create Report</Title>
            <Grid size={{ xs: 12, md: 8 }}>
                <Stack spacing={2}>
                    {canReport && <ReportCreateForm />}
                    {!canReport && (
                        <ContainerWithHeader title={'Permission Denied'}>
                            <Typography variant={'body1'} padding={2}>
                                You are unable to report players while you are currently banned/muted.
                            </Typography>
                            <ButtonGroup sx={{ padding: 2 }}>
                                <ButtonLink
                                    variant={'contained'}
                                    color={'primary'}
                                    to={'/ban/$ban_id'}
                                    params={{ ban_id: profile.ban_id.toString() }}
                                >
                                    Appeal Ban
                                </ButtonLink>
                            </ButtonGroup>
                        </ContainerWithHeader>
                    )}
                    <ContainerWithHeader title={'Your Report History'} iconLeft={<HistoryIcon />}>
                        {isLoading ? (
                            <LoadingPlaceholder />
                        ) : (
                            <UserReportHistory history={logs ?? []} isLoading={isLoading} />
                        )}
                    </ContainerWithHeader>
                </Stack>
            </Grid>
            <Grid size={{ xs: 12, md: 4 }}>
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

const UserReportHistory = ({ history, isLoading }: { history: ReportWithAuthor[]; isLoading: boolean }) => {
    const [pagination, setPagination] = useState(initPagination(0, RowsPerPage.Ten));
    const [sorting] = useState<SortingState>([{ id: 'report_id', desc: true }]);

    const columns = [
        columnHelper.accessor('report_status', {
            header: 'Status',
            size: 150,
            cell: (info) => {
                return (
                    <Stack direction={'row'} spacing={1}>
                        <ReportStatusIcon reportStatus={info.getValue()} />
                        <Typography variant={'body1'}>{reportStatusString(info.getValue())}</Typography>
                    </Stack>
                );
            }
        }),
        columnHelper.accessor('subject', {
            header: 'Player',
            cell: (info) => (
                <PersonCell
                    steam_id={info.row.original.subject.steam_id}
                    personaname={info.row.original.subject.personaname}
                    avatar_hash={info.row.original.subject.avatarhash}
                />
            )
        }),
        columnHelper.accessor('report_id', {
            header: 'View',
            size: 30,
            cell: (info) => (
                <ButtonGroup>
                    <IconButtonLink
                        color={'primary'}
                        to={`/report/$reportId`}
                        params={{ reportId: String(info.getValue()) }}
                    >
                        <Tooltip title={'View'}>
                            <VisibilityIcon />
                        </Tooltip>
                    </IconButtonLink>
                </ButtonGroup>
            )
        })
    ];

    const table = useReactTable({
        data: history,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: false,
        autoResetPageIndex: true,
        getPaginationRowModel: getPaginationRowModel(),
        getSortedRowModel: getSortedRowModel(),
        onPaginationChange: setPagination,
        state: {
            pagination,
            sorting
        }
    });

    return (
        <>
            <DataTable table={table} isLoading={isLoading} />
            <PaginatorLocal
                onRowsChange={(rows) => {
                    setPagination((prev) => {
                        return { ...prev, pageSize: rows };
                    });
                }}
                onPageChange={(page) => {
                    setPagination((prev) => {
                        return { ...prev, pageIndex: page };
                    });
                }}
                count={history?.length ?? 0}
                rows={pagination.pageSize}
                page={pagination.pageIndex}
            />
        </>
    );
};

export const ReportCreateForm = (): JSX.Element => {
    const { demo_id, steam_id, person_message_id } = Route.useSearch();
    const [validatedProfile] = useState<PlayerProfile>();
    const { sendFlash, sendError } = useUserFlashCtx();

    const mutation = useMutation({
        mutationFn: async (variables: CreateReportRequest) => {
            return await apiCreateReport(variables);
        },
        onSuccess: async (data) => {
            mdEditorRef.current?.setMarkdown('');
            await navigate({ to: '/report/$reportId', params: { reportId: String(data.report_id) } });
            sendFlash('success', 'Created report successfully');
        },
        onError: sendError
    });

    const navigate = useNavigate();

    const form = useForm({
        onSubmit: ({ value }) => {
            mutation.mutate({
                demo_id: value.demo_id ?? 0,
                target_id: steam_id ?? validatedProfile?.player.steam_id ?? '',
                demo_tick: value.demo_tick,
                reason: value.reason,
                reason_text: value.reason_text,
                description: value.body_md,
                person_message_id: value.person_message_id
            });
        },
        // validators: {
        //     onChange: validationSchema
        // },
        defaultValues: {
            body_md: '',
            demo_id: demo_id ?? undefined,
            demo_tick: 0,
            person_message_id: person_message_id ?? 0,
            steam_id: steam_id ?? '',
            reason: person_message_id ? BanReason.Language : BanReason.Cheating,
            reason_text: ''
        }
    });

    return (
        <ContainerWithHeader title={'Create New Report'} iconLeft={<EditNotificationsIcon />} spacing={2} marginTop={3}>
            <form
                id={'reportForm'}
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <Grid container spacing={2}>
                    <Grid size={{ xs: 12 }}>
                        <form.Field
                            name={'steam_id'}
                            children={({ handleChange, handleBlur }) => {
                                return (
                                    <SteamIDField
                                        disabled={Boolean(steam_id)}
                                        handleBlur={handleBlur}
                                        handleChange={handleChange}
                                        fullwidth={true}
                                        label={'SteamID'}
                                    />
                                );
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 6 }}>
                        <form.Field
                            name={'reason'}
                            validators={{
                                onChange: z.nativeEnum(BanReason, { message: 'Invalid ban reason' })
                            }}
                            children={({ state, handleChange, handleBlur }) => {
                                return (
                                    <>
                                        <FormControl fullWidth>
                                            <InputLabel id="serverSelectLabel">Reason</InputLabel>
                                            <Select
                                                variant={'outlined'}
                                                fullWidth
                                                value={state.value}
                                                label="Servers"
                                                onChange={(e) => {
                                                    handleChange(Number(e.target.value));
                                                }}
                                                onBlur={handleBlur}
                                                error={state.meta.errors.length > 0}
                                            >
                                                {banReasonsCollection.map((r) => (
                                                    <MenuItem value={r} key={`reason-${r}`}>
                                                        {BanReasons[r]}
                                                    </MenuItem>
                                                ))}
                                            </Select>
                                            {state.meta.errors.length > 0 && <FormHelperText>Error</FormHelperText>}
                                        </FormControl>
                                    </>
                                );
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 6 }}>
                        <form.Field
                            name={'reason_text'}
                            validators={{
                                onChangeListenTo: ['reason'],
                                onChange: ({ value, fieldApi }) => {
                                    if (fieldApi.form.getFieldValue('reason') == BanReason.Custom) {
                                        return z.string().min(2, { message: 'Must enter custom reason' }).parse(value);
                                    }

                                    return z.string().parse(value);
                                }
                            }}
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
                    {Boolean(demo_id) && (
                        <>
                            <Grid size={{ xs: 6 }}>
                                <form.Field
                                    name={'demo_id'}
                                    validators={{ onChange: z.number({ coerce: true }).optional() }}
                                    children={({ state, handleChange, handleBlur }) => {
                                        return (
                                            <TextField
                                                disabled={Boolean(demo_id)}
                                                fullWidth
                                                label="Demo ID"
                                                value={state.value}
                                                onChange={(e) => handleChange(Number(e.target.value))}
                                                onBlur={handleBlur}
                                                variant="outlined"
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 6 }}>
                                <form.Field
                                    // validators={{ onChange: z.number({ coerce: true }).min(0).optional() }}
                                    name={'demo_tick'}
                                    children={({ state, handleChange, handleBlur }) => {
                                        return (
                                            <TextField
                                                disabled={!demo_id}
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
                        </>
                    )}
                    {person_message_id != undefined && person_message_id > 0 && (
                        <Grid size={{ md: 12 }}>
                            <PlayerMessageContext playerMessageId={person_message_id} padding={5} />
                        </Grid>
                    )}
                    <Grid size={{ xs: 12 }}>
                        <form.Field
                            name={'body_md'}
                            validators={{ onChange: z.string().min(10, 'Message must be at least 10 characters.') }}
                            children={(props) => {
                                return <MarkdownField {...props} label={'Message (Markdown)'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <form.Subscribe
                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                            children={([canSubmit, isSubmitting]) => {
                                return <Buttons canSubmit={canSubmit} isSubmitting={isSubmitting} reset={form.reset} />;
                            }}
                        />
                    </Grid>
                </Grid>
            </form>
        </ContainerWithHeader>
    );
};
