import { useState, JSX, SyntheticEvent } from 'react';
import DescriptionIcon from '@mui/icons-material/Description';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import LanIcon from '@mui/icons-material/Lan';
import MessageIcon from '@mui/icons-material/Message';
import ReportIcon from '@mui/icons-material/Report';
import ReportGmailerrorredIcon from '@mui/icons-material/ReportGmailerrorred';
import TabContext from '@mui/lab/TabContext';
import TabList from '@mui/lab/TabList';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { useForm } from '@tanstack/react-form';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouteContext } from '@tanstack/react-router';
import {
    apiCreateReportMessage,
    apiDeleteReportMessage,
    apiGetBansSteam,
    apiGetConnections,
    apiGetMessages,
    apiGetReportMessages,
    PermissionLevel,
    Report
} from '../api';
import { RowsPerPage } from '../util/table.ts';
import { BanHistoryTable } from './BanHistoryTable.tsx';
import { ChatTable } from './ChatTable.tsx';
import { ContainerWithHeader } from './ContainerWithHeader';
import { IPHistoryTable } from './IPHistoryTable.tsx';
import { MarkDownRenderer } from './MarkdownRenderer';
import { PaginatorLocal } from './PaginatorLocal.tsx';
import { PlayerMessageContext } from './PlayerMessageContext';
import { ReportMessageView } from './ReportMessageView';
import { SourceBansList } from './SourceBansList';
import { TabPanel } from './TabPanel';
import { MDBodyField } from './_formik/MDBodyField.tsx';
import { Buttons } from './field/Buttons.tsx';

const messagesQueryOptions = (reportId: number) => ({
    queryKey: ['reportMessages', { reportID: reportId }],
    queryFn: async () => {
        return (await apiGetReportMessages(reportId)) ?? [];
    }
});

export const ReportViewComponent = ({ report }: { report: Report }): JSX.Element => {
    const theme = useTheme();
    const queryClient = useQueryClient();
    // const { data: messagesServer } = useReportMessages(report.report_id);
    // const [deletedMessages, setDeletedMessages] = useState<number[]>([]);
    const [value, setValue] = useState<number>(0);
    const { hasPermission } = useRouteContext({ from: '/_auth/report/$reportId' });

    const [chatPagination, setChatPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const [connectionPagination, setConnectionPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const { data: connections, isLoading: isLoadingConnections } = useQuery({
        queryKey: ['reportConnectionHist', { steamId: report.target_id }],
        queryFn: async () => {
            return await apiGetConnections({
                limit: 1000,
                offset: 0,
                order_by: 'person_connection_id',
                desc: true,
                source_id: report.target_id
            });
        }
    });

    const { data: chat, isLoading: isLoadingChat } = useQuery({
        queryKey: ['reportChat', { steamId: report.target_id }],
        queryFn: async () => {
            return await apiGetMessages({
                personaname: '',
                query: '',
                source_id: report.target_id,
                limit: 2500,
                offset: 0,
                order_by: 'person_message_id',
                desc: true,
                flagged_only: false
            });
        }
    });

    const { data: messages, isLoading: isLoadingMessages } = useQuery(messagesQueryOptions(report.report_id));

    const { data: bans, isLoading: isLoadingBans } = useQuery({
        queryKey: ['reportBanHistory', { steamId: report.target_id }],
        queryFn: async () => {
            return await apiGetBansSteam({
                limit: 100,
                offset: 0,
                order_by: 'ban_id',
                desc: true,
                target_id: report.target_id,
                deleted: true
            });
        }
    });

    const handleChange = (_: SyntheticEvent, newValue: number) => {
        setValue(newValue);
    };

    const createMessageMutation = useMutation({
        mutationFn: async ({ body_md }: { body_md: string }) => {
            return await apiCreateReportMessage(report.report_id, body_md);
        },
        onSuccess: (message) => {
            queryClient.setQueryData(messagesQueryOptions(report.report_id).queryKey, [...(messages ?? []), message]);
            reset();
        }
    });

    const deleteMessageMutation = useMutation({
        mutationFn: async ({ message_id }: { message_id: number }) => {
            return await apiDeleteReportMessage(message_id);
        },
        onSuccess: (_, { message_id }) => {
            queryClient.setQueryData(
                messagesQueryOptions(report.report_id).queryKey,
                (messages ?? []).filter((m) => m.report_message_id != message_id)
            );
        }
    });

    const onDelete = async (message_id: number) => {
        deleteMessageMutation.mutate({ message_id });
    };

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            createMessageMutation.mutate(value);
        },
        defaultValues: {
            body_md: ''
        }
    });

    return (
        <Grid container>
            <Grid xs={12}>
                <TabContext value={value}>
                    <Stack spacing={2}>
                        <ContainerWithHeader title={'Report Overview'} iconLeft={<ReportIcon />}>
                            <Box
                                sx={{
                                    borderBottom: 1,
                                    borderColor: 'divider',
                                    backgroundColor: theme.palette.background.paper
                                }}
                            >
                                <TabList variant={'fullWidth'} onChange={handleChange} aria-label="ReportCreatePage detail tabs">
                                    <Tab label="Description" icon={<DescriptionIcon />} iconPosition={'start'} />
                                    {hasPermission(PermissionLevel.Moderator) && (
                                        <Tab sx={{ height: 20 }} label={`Chat Logs`} icon={<MessageIcon />} iconPosition={'start'} />
                                    )}
                                    {hasPermission(PermissionLevel.Moderator) && (
                                        <Tab label={`Connections`} icon={<LanIcon />} iconPosition={'start'} />
                                    )}
                                    {hasPermission(PermissionLevel.Moderator) && (
                                        <Tab
                                            label={`Ban History ${bans ? `(${bans.data.length})` : ''}`}
                                            icon={<ReportGmailerrorredIcon />}
                                            iconPosition={'start'}
                                        />
                                    )}
                                </TabList>
                            </Box>

                            <TabPanel value={value} index={0}>
                                {report && (
                                    <Box minHeight={300}>
                                        <MarkDownRenderer body_md={report.description} />
                                    </Box>
                                )}
                            </TabPanel>

                            <TabPanel value={value} index={1}>
                                <Box minHeight={300}>
                                    <ChatTable
                                        messages={chat ?? []}
                                        isLoading={isLoadingChat}
                                        manualPaging={false}
                                        pagination={chatPagination}
                                        setPagination={setChatPagination}
                                    />
                                    <PaginatorLocal
                                        onRowsChange={(rows) => {
                                            setChatPagination((prev) => {
                                                return { ...prev, pageSize: rows };
                                            });
                                        }}
                                        onPageChange={(page) => {
                                            setChatPagination((prev) => {
                                                return { ...prev, pageIndex: page };
                                            });
                                        }}
                                        count={chat?.length ?? 0}
                                        rows={chatPagination.pageSize}
                                        page={chatPagination.pageIndex}
                                    />
                                </Box>
                            </TabPanel>
                            <TabPanel value={value} index={2}>
                                <Box minHeight={300}>
                                    <IPHistoryTable
                                        connections={connections ?? { data: [], count: 0 }}
                                        isLoading={isLoadingConnections}
                                        manualPaging={false}
                                        pagination={connectionPagination}
                                        setPagination={setConnectionPagination}
                                    />
                                    <PaginatorLocal
                                        onRowsChange={(rows) => {
                                            setConnectionPagination((prev) => {
                                                return { ...prev, pageSize: rows };
                                            });
                                        }}
                                        onPageChange={(page) => {
                                            setConnectionPagination((prev) => {
                                                return { ...prev, pageIndex: page };
                                            });
                                        }}
                                        count={connections?.data?.length ?? 0}
                                        rows={connectionPagination.pageSize}
                                        page={connectionPagination.pageIndex}
                                    />
                                </Box>
                            </TabPanel>
                            <TabPanel value={value} index={3}>
                                <Box
                                    minHeight={300}
                                    style={{
                                        display: value == 3 ? 'block' : 'none'
                                    }}
                                >
                                    <BanHistoryTable bans={bans ?? { data: [], count: 0 }} isLoading={isLoadingBans} />
                                </Box>
                            </TabPanel>
                        </ContainerWithHeader>
                        {report.demo_name != '' && (
                            <Paper>
                                <Stack direction={'row'} padding={1}>
                                    <Typography padding={2} variant={'button'} alignContent={'center'}>
                                        Demo&nbsp;Info
                                    </Typography>
                                    <Typography padding={2} variant={'body1'} alignContent={'center'}>
                                        Tick:&nbsp;{report.demo_tick}
                                    </Typography>
                                    <Button
                                        fullWidth
                                        startIcon={<FileDownloadIcon />}
                                        component={Link}
                                        variant={'text'}
                                        href={`${window.gbans.asset_url}/${window.gbans.bucket_demo}/${report.demo_name}`}
                                        color={'primary'}
                                    >
                                        {report.demo_name}
                                    </Button>
                                </Stack>
                            </Paper>
                        )}

                        {report.person_message_id > 0 && (
                            <ContainerWithHeader title={'Message Context'}>
                                <PlayerMessageContext playerMessageId={report.person_message_id} padding={4} />
                            </ContainerWithHeader>
                        )}

                        {hasPermission(PermissionLevel.Moderator) && <SourceBansList steam_id={report.source_id} is_reporter={true} />}

                        {hasPermission(PermissionLevel.Moderator) && <SourceBansList steam_id={report.target_id} is_reporter={false} />}

                        {!isLoadingMessages &&
                            messages &&
                            messages.map((m) => (
                                <ReportMessageView onDelete={onDelete} message={m} key={`report-msg-${m.report_message_id}`} />
                            ))}
                        <Paper elevation={1}>
                            <form
                                onSubmit={async (e) => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                    await handleSubmit();
                                }}
                            >
                                <Grid container>
                                    <Grid xs={12}>
                                        <Field
                                            name={'body_md'}
                                            children={(props) => {
                                                return <MDBodyField {...props} label={'Message'} fullwidth={true} />;
                                            }}
                                        />
                                    </Grid>
                                    <Grid xs={12} mdOffset="auto">
                                        <Subscribe
                                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                                            children={([canSubmit, isSubmitting]) => (
                                                <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />
                                            )}
                                        />
                                    </Grid>
                                </Grid>
                            </form>
                        </Paper>
                    </Stack>
                </TabContext>
            </Grid>
        </Grid>
    );
};
