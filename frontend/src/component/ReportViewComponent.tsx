import { useState, JSX, SyntheticEvent } from 'react';
import DescriptionIcon from '@mui/icons-material/Description';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import LanIcon from '@mui/icons-material/Lan';
import MessageIcon from '@mui/icons-material/Message';
import QuickreplyIcon from '@mui/icons-material/Quickreply';
import ReportIcon from '@mui/icons-material/Report';
import ReportGmailerrorredIcon from '@mui/icons-material/ReportGmailerrorred';
import VideocamIcon from '@mui/icons-material/Videocam';
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
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import {
    apiCreateReportMessage,
    apiGetBansSteam,
    apiGetConnections,
    apiGetMessages,
    PermissionLevel,
    ReportWithAuthor
} from '../api';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { reportMessagesQueryOptions } from '../queries/reportMessages.ts';
import { RowsPerPage } from '../util/table.ts';
import { BanHistoryTable } from './BanHistoryTable.tsx';
import { ChatTable } from './ChatTable.tsx';
import { ContainerWithHeader } from './ContainerWithHeader';
import { ContainerWithHeaderAndButtons } from './ContainerWithHeaderAndButtons.tsx';
import { IPHistoryTable } from './IPHistoryTable.tsx';
import { MarkDownRenderer } from './MarkdownRenderer';
import { PaginatorLocal } from './PaginatorLocal.tsx';
import { PlayerMessageContext } from './PlayerMessageContext';
import { ReportMessageView } from './ReportMessageView';
import { SourceBansList } from './SourceBansList';
import { TabPanel } from './TabPanel';
import { Buttons } from './field/Buttons.tsx';
import { MarkdownField, mdEditorRef } from './field/MarkdownField.tsx';

export const ReportViewComponent = ({ report }: { report: ReportWithAuthor }): JSX.Element => {
    const theme = useTheme();
    const queryClient = useQueryClient();
    const { sendFlash } = useUserFlashCtx();
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

    const { data: messages, isLoading: isLoadingMessages } = useQuery(reportMessagesQueryOptions(report.report_id));

    const { data: bans, isLoading: isLoadingBans } = useQuery({
        queryKey: ['reportBanHistory', { steamId: report.target_id }],
        queryFn: async () => {
            const bans = await apiGetBansSteam({ target_id: report.target_id });

            return bans.filter((b) => b.target_id == report.target_id);
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
            queryClient.setQueryData(reportMessagesQueryOptions(report.report_id).queryKey, [
                ...(messages ?? []),
                message
            ]);
            mdEditorRef.current?.setMarkdown('');
            reset();
            sendFlash('success', 'Created message successfully');
        },
        onError: (error) => {
            sendFlash('error', `Failed to create message: ${error}`);
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            createMessageMutation.mutate(value);
        },
        validatorAdapter: zodValidator,
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
                                <TabList
                                    variant={'fullWidth'}
                                    onChange={handleChange}
                                    aria-label="ReportCreatePage detail tabs"
                                >
                                    <Tab label="Description" icon={<DescriptionIcon />} iconPosition={'start'} />
                                    {hasPermission(PermissionLevel.Moderator) && (
                                        <Tab
                                            sx={{ height: 20 }}
                                            label={`Chat Logs`}
                                            icon={<MessageIcon />}
                                            iconPosition={'start'}
                                        />
                                    )}
                                    {hasPermission(PermissionLevel.Moderator) && (
                                        <Tab label={`Connections`} icon={<LanIcon />} iconPosition={'start'} />
                                    )}
                                    {hasPermission(PermissionLevel.Moderator) && (
                                        <Tab
                                            label={`Ban History ${bans ? `(${bans.length})` : ''}`}
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
                                    <BanHistoryTable bans={bans ?? []} isLoading={isLoadingBans} />
                                </Box>
                            </TabPanel>
                        </ContainerWithHeader>
                        {report.demo.demo_id > 0 && (
                            <ContainerWithHeaderAndButtons
                                title={`Demo Details: ${report.demo.title}`}
                                iconLeft={<VideocamIcon />}
                                buttons={[
                                    <Button
                                        variant={'contained'}
                                        fullWidth
                                        key={'demo_download'}
                                        startIcon={<FileDownloadIcon />}
                                        component={Link}
                                        href={`/asset/${report.demo_id}`}
                                        color={'success'}
                                    >
                                        Download
                                    </Button>
                                ]}
                            >
                                <Grid container padding={2}>
                                    <Grid xs={4}>
                                        <Typography>Map:&nbsp;{report.demo.map_name}</Typography>
                                    </Grid>
                                    <Grid xs={4}>
                                        <Typography>Server:&nbsp;{report.demo.server_name_short}</Typography>
                                    </Grid>
                                    <Grid xs={2}>
                                        <Typography>Tick:&nbsp;{report.demo_tick}</Typography>
                                    </Grid>
                                    <Grid xs={2}>
                                        <Typography>ID:&nbsp;{report.demo_id}</Typography>
                                    </Grid>
                                </Grid>
                            </ContainerWithHeaderAndButtons>
                        )}

                        {report.person_message_id > 0 && (
                            <ContainerWithHeader title={'Message Context'} iconLeft={<QuickreplyIcon />}>
                                <PlayerMessageContext playerMessageId={report.person_message_id} padding={4} />
                            </ContainerWithHeader>
                        )}

                        {hasPermission(PermissionLevel.Moderator) && (
                            <SourceBansList steam_id={report.source_id} is_reporter={true} />
                        )}

                        {hasPermission(PermissionLevel.Moderator) && (
                            <SourceBansList steam_id={report.target_id} is_reporter={false} />
                        )}

                        {!isLoadingMessages &&
                            messages &&
                            messages.map((m) => (
                                <ReportMessageView message={m} key={`report-msg-${m.report_message_id}`} />
                            ))}
                        <Paper elevation={1}>
                            <form
                                onSubmit={async (e) => {
                                    e.preventDefault();
                                    e.stopPropagation();
                                    await handleSubmit();
                                }}
                            >
                                <Grid container spacing={2} padding={1}>
                                    <Grid xs={12}>
                                        <Field
                                            name={'body_md'}
                                            validators={{
                                                onChange: z.string().min(2)
                                            }}
                                            children={(props) => {
                                                return <MarkdownField {...props} label={'Message'} fullwidth={true} />;
                                            }}
                                        />
                                    </Grid>
                                    <Grid xs={12} mdOffset="auto">
                                        <Subscribe
                                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                                            children={([canSubmit, isSubmitting]) => (
                                                <Buttons
                                                    reset={reset}
                                                    canSubmit={canSubmit}
                                                    isSubmitting={isSubmitting}
                                                />
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
