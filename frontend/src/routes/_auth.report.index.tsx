import { JSX, useMemo, useState } from 'react';
import EditNotificationsIcon from '@mui/icons-material/EditNotifications';
import HistoryIcon from '@mui/icons-material/History';
import InfoIcon from '@mui/icons-material/Info';
import VisibilityIcon from '@mui/icons-material/Visibility';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import Link from '@mui/material/Link';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
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
import { apiCreateReport, apiGetUserReports } from '../api';
import { ButtonLink } from '../component/ButtonLink.tsx';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { IconButtonLink } from '../component/IconButtonLink.tsx';
import { LoadingPlaceholder } from '../component/LoadingPlaceholder.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { PlayerMessageContext } from '../component/PlayerMessageContext.tsx';
import { ReportStatusIcon } from '../component/ReportStatusIcon.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { Title } from '../component/Title';
import { mdEditorRef } from '../component/form/field/MarkdownField.tsx';
import { PaginatorLocal } from '../component/forum/PaginatorLocal.tsx';
import { DataTable } from '../component/table/DataTable.tsx';
import { useAppForm } from '../contexts/formContext.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { BanReason, BanReasons, banReasonsReportCollection } from '../schema/bans.ts';
import {
    CreateReportRequest,
    reportStatusString,
    ReportWithAuthor,
    schemaCreateReportRequest
} from '../schema/report.ts';
import { commonTableSearchSchema, initPagination, RowsPerPage } from '../util/table.ts';
import { emptyOrNullString } from '../util/types.ts';

const searchSchemaReport = z.object({
    ...commonTableSearchSchema,
    rows: z.number().optional(),
    sortColumn: z.enum(['report_status', 'created_on']).optional(),
    steam_id: z.string().optional(),
    demo_id: z.number({ coerce: true }).optional(),
    person_message_id: z.number().optional()
});

export const Route = createFileRoute('/_auth/report/')({
    component: ReportCreate,
    validateSearch: (search) => searchSchemaReport.parse(search)
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
                    {canReport ? (
                        <ReportCreateForm />
                    ) : (
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
    const { sendFlash, sendError } = useUserFlashCtx();
    const [isCustom, setIsCustom] = useState(false);

    const defaultValues: z.input<typeof schemaCreateReportRequest> = {
        description: '',
        demo_id: demo_id ?? 0,
        demo_tick: 0,
        person_message_id: person_message_id ?? 0,
        target_id: steam_id ?? '',
        reason: person_message_id ? BanReason.Language : BanReason.Cheating,
        reason_text: ''
    };

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

    const form = useAppForm({
        onSubmit: ({ value }) => {
            mutation.mutate({
                demo_id: value?.demo_id ?? 0,
                target_id: value.target_id,
                demo_tick: value.demo_tick,
                reason: value.reason,
                reason_text: value.reason_text,
                description: value.description,
                person_message_id: value.person_message_id
            });
        },
        validators: {
            onSubmit: schemaCreateReportRequest
        },
        defaultValues
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
                        <form.AppField
                            name={'target_id'}
                            children={(field) => {
                                return <field.SteamIDField disabled={Boolean(steam_id)} label={'SteamID'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 6 }}>
                        <form.AppField
                            name={'reason'}
                            children={(field) => {
                                return (
                                    <field.SelectField
                                        label={'Ban Reason'}
                                        items={banReasonsReportCollection}
                                        handleChange={(value) => {
                                            setIsCustom(value == BanReason.Custom);
                                            field.handleChange(value);
                                        }}
                                        renderItem={(r) => {
                                            return (
                                                <MenuItem value={r} key={`reason-${r}`}>
                                                    {BanReasons[r]}
                                                </MenuItem>
                                            );
                                        }}
                                    />
                                );
                            }}
                        />
                    </Grid>

                    <Grid size={{ xs: 6 }}>
                        <form.AppField
                            name={'reason_text'}
                            validators={{
                                onChangeListenTo: ['reason', 'reason_text'],
                                onChange: ({ value, fieldApi }) => {
                                    if (!emptyOrNullString(value)) {
                                        if (BanReason.Custom !== fieldApi.form.getFieldValue('reason')) {
                                            return 'Reason must be set to custom';
                                        }
                                        fieldApi.form.setFieldValue('reason', () => {
                                            return BanReason.Custom;
                                        });
                                        return undefined;
                                    }
                                }
                            }}
                            children={(field) => {
                                return (
                                    <field.TextField
                                        fullWidth
                                        disabled={!isCustom}
                                        label="Custom Reason"
                                        helperText={'You must set the reason to Custom to use this field'}
                                    />
                                );
                            }}
                        />
                    </Grid>
                    {Boolean(demo_id) && (
                        <>
                            <Grid size={{ xs: 6 }}>
                                <form.AppField
                                    name={'demo_id'}
                                    children={(field) => {
                                        return <field.TextField disabled={Boolean(demo_id)} label="Demo ID" />;
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 6 }}>
                                <form.AppField
                                    name={'demo_tick'}
                                    children={(field) => {
                                        return (
                                            <field.TextField disabled={!demo_id} label="Demo Tick" variant="outlined" />
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
                        <form.AppField
                            name={'description'}
                            children={(field) => {
                                return <field.MarkdownField label={'Message (Markdown)'} />;
                            }}
                        />
                    </Grid>
                    <Grid size={{ xs: 12 }}>
                        <form.AppForm>
                            <ButtonGroup>
                                <form.ResetButton />
                                <form.SubmitButton />
                            </ButtonGroup>
                        </form.AppForm>
                    </Grid>
                </Grid>
            </form>
        </ContainerWithHeader>
    );
};
