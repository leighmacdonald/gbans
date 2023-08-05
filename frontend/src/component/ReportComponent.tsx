import React, { useCallback, useEffect, useState, JSX } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import Paper from '@mui/material/Paper';
import Box from '@mui/material/Box';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import useTheme from '@mui/material/styles/useTheme';
import {
    apiCreateReportMessage,
    apiDeleteReportMessage,
    apiGetPersonConnections,
    apiGetPersonMessages,
    apiGetReportMessages,
    apiUpdateReportMessage,
    BanReasons,
    IAPIBanRecordProfile,
    PermissionLevel,
    PersonConnection,
    PersonMessage,
    Report,
    ReportMessagesResponse,
    UserMessage
} from '../api';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { logErr } from '../util/errors';
import { renderMarkdown } from '../api/wiki';
import { MDEditor } from './MDEditor';
import { DataTable, RowsPerPage } from './DataTable';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { RenderedMarkdownBox } from './RenderedMarkdownBox';
import { UserMessageView } from './UserMessageView';
import { TabPanel } from './TabPanel';
import { PersonMessageTable } from './PersonMessageTable';
import Link from '@mui/material/Link';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import Button from '@mui/material/Button';
import { SourceBansList } from './SourceBansList';
import { PlayerMessageContext } from './PlayerMessageContext';
import { ContainerWithHeader } from './ContainerWithHeader';

interface ReportComponentProps {
    report: Report;
    banHistory: IAPIBanRecordProfile[];
}

export const ReportComponent = ({
    report,
    banHistory
}: ReportComponentProps): JSX.Element => {
    const theme = useTheme();
    const [messages, setMessages] = useState<ReportMessagesResponse[]>([]);
    const [connections, setConnections] = useState<PersonConnection[]>([]);
    const [chatHistory, setChatHistory] = useState<PersonMessage[]>([]);

    const [value, setValue] = React.useState<number>(0);
    const { currentUser } = useCurrentUserCtx();
    const { sendFlash } = useUserFlashCtx();

    const handleChange = (_: React.SyntheticEvent, newValue: number) => {
        setValue(newValue);
    };

    const loadMessages = useCallback(() => {
        apiGetReportMessages(report.report_id)
            .then((response) => {
                setMessages(response.result || []);
            })
            .catch(logErr);
    }, [report.report_id]);

    const onSave = useCallback(
        (message: string, onSuccess?: () => void) => {
            apiCreateReportMessage(report.report_id, message)
                .then((response) => {
                    if (!response.status || !response.result) {
                        sendFlash('error', 'Failed to save report message');
                        return;
                    }
                    setMessages([
                        ...messages,
                        { author: currentUser, message: response.result }
                    ]);
                    onSuccess && onSuccess();
                })
                .catch(logErr);
        },
        [report.report_id, messages, currentUser, sendFlash]
    );

    const onEdit = useCallback(
        (message: UserMessage) => {
            apiUpdateReportMessage(message.message_id, message.contents)
                .then(() => {
                    sendFlash('success', 'Updated message successfully');
                    loadMessages();
                })
                .catch(logErr);
        },
        [loadMessages, sendFlash]
    );

    const onDelete = useCallback(
        (message_id: number) => {
            apiDeleteReportMessage(message_id)
                .then((response) => {
                    if (!response.status) {
                        sendFlash('error', 'Failed to delete message');
                        return;
                    }
                    sendFlash('success', 'Deleted message successfully');
                    loadMessages();
                })
                .catch(logErr);
        },
        [loadMessages, sendFlash]
    );

    useEffect(() => {
        loadMessages();
    }, [loadMessages, report]);

    useEffect(() => {
        apiGetPersonConnections(report.target_id)
            .then((response) => {
                setConnections(response.result || []);
            })
            .catch(logErr);
    }, [report]);

    useEffect(() => {
        apiGetPersonMessages(report.target_id)
            .then((response) => {
                setChatHistory(response.result || []);
            })
            .catch(logErr);
    }, [report]);

    return (
        <Grid container>
            <Grid xs={12}>
                <Stack spacing={2}>
                    <Paper elevation={1} sx={{ width: '100%', minHeight: 400 }}>
                        <Box
                            sx={{
                                borderBottom: 1,
                                borderColor: 'divider',
                                backgroundColor: theme.palette.background.paper
                            }}
                        >
                            <Tabs
                                value={value}
                                onChange={handleChange}
                                aria-label="ReportCreatePage detail tabs"
                            >
                                <Tab label="Description" />
                                {currentUser.permission_level >=
                                    PermissionLevel.Moderator && (
                                    <Tab
                                        label={`Chat Logs (${chatHistory.length})`}
                                    />
                                )}
                                {currentUser.permission_level >=
                                    PermissionLevel.Moderator && (
                                    <Tab
                                        label={`Connections (${connections.length})`}
                                    />
                                )}
                                {currentUser.permission_level >=
                                    PermissionLevel.Moderator && (
                                    <Tab
                                        label={`Ban History (${banHistory.length})`}
                                    />
                                )}
                            </Tabs>
                        </Box>

                        <TabPanel value={value} index={0}>
                            {report && (
                                <RenderedMarkdownBox
                                    bodyHTML={renderMarkdown(
                                        report.description
                                    )}
                                    readonly={true}
                                    setEditMode={() => {
                                        return false;
                                    }}
                                />
                            )}
                        </TabPanel>

                        <TabPanel value={value} index={1}>
                            <PersonMessageTable messages={chatHistory} />
                        </TabPanel>
                        <TabPanel value={value} index={2}>
                            <DataTable
                                columns={[
                                    {
                                        label: 'Created',
                                        tooltip: 'Created On',
                                        sortKey: 'created_on',
                                        sortType: 'date',
                                        align: 'left',
                                        width: '150px'
                                    },
                                    {
                                        label: 'Name',
                                        tooltip: 'Name',
                                        sortKey: 'persona_name',
                                        sortType: 'string',
                                        align: 'left',
                                        width: '150px',
                                        queryValue: (row) => row.persona_name
                                    },
                                    {
                                        label: 'IP Address',
                                        tooltip: 'IP Address',
                                        sortKey: 'ip_addr',
                                        sortType: 'string',
                                        align: 'left',
                                        queryValue: (row) => row.ip_addr
                                    }
                                ]}
                                defaultSortColumn={'created_on'}
                                rowsPerPage={RowsPerPage.TwentyFive}
                                rows={connections}
                            />
                        </TabPanel>
                        <TabPanel value={value} index={3}>
                            <DataTable
                                columns={[
                                    {
                                        label: 'Created',
                                        tooltip: 'Created On',
                                        sortKey: 'created_on',
                                        sortType: 'date',
                                        sortable: true,
                                        align: 'left',
                                        width: '150px'
                                    },
                                    {
                                        label: 'Expires',
                                        tooltip: 'Expires',
                                        sortKey: 'valid_until',
                                        sortType: 'date',
                                        sortable: true,
                                        align: 'left'
                                    },
                                    {
                                        label: 'Ban Author',
                                        tooltip: 'Ban Author',
                                        sortKey: 'source_id',
                                        sortType: 'string',
                                        align: 'left',
                                        width: '150px',
                                        renderer: (row) => (
                                            <Typography variant={'body1'}>
                                                {row.source_id}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'Reason',
                                        tooltip: 'Reason',
                                        sortKey: 'reason',
                                        sortable: true,
                                        sortType: 'string',
                                        align: 'left',
                                        renderer: (row) => (
                                            <Typography variant={'body1'}>
                                                {BanReasons[row.reason]}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'Custom',
                                        tooltip: 'Custom Reason',
                                        sortKey: 'reason_text',
                                        sortType: 'string',
                                        align: 'left'
                                    },
                                    {
                                        label: 'Unban Reason',
                                        tooltip: 'Unban Reason',
                                        sortKey: 'unban_reason_text',
                                        sortType: 'string',
                                        align: 'left'
                                    }
                                ]}
                                defaultSortColumn={'created_on'}
                                rowsPerPage={RowsPerPage.TwentyFive}
                                rows={banHistory}
                            />
                        </TabPanel>
                    </Paper>
                    {report.demo_name != '' && (
                        <Paper>
                            <Stack direction={'row'} padding={1}>
                                <Typography
                                    padding={2}
                                    variant={'button'}
                                    alignContent={'center'}
                                >
                                    Demo&nbsp;Info
                                </Typography>
                                <Typography
                                    padding={2}
                                    variant={'body1'}
                                    alignContent={'center'}
                                >
                                    Tick:&nbsp;{report.demo_tick}
                                </Typography>
                                <Button
                                    fullWidth
                                    startIcon={<FileDownloadIcon />}
                                    component={Link}
                                    variant={'text'}
                                    href={`/demos/${report.demo_id}`}
                                    color={'primary'}
                                >
                                    {report.demo_name}
                                </Button>
                            </Stack>
                        </Paper>
                    )}

                    {report.person_message_id > 0 && (
                        <ContainerWithHeader title={'Message Context'}>
                            <PlayerMessageContext
                                playerMessageId={report.person_message_id}
                                padding={4}
                            />
                        </ContainerWithHeader>
                    )}

                    {currentUser.permission_level >=
                        PermissionLevel.Moderator && (
                        <SourceBansList
                            steam_id={report.source_id}
                            is_reporter={true}
                        />
                    )}

                    {currentUser.permission_level >=
                        PermissionLevel.Moderator && (
                        <SourceBansList
                            steam_id={report.target_id}
                            is_reporter={false}
                        />
                    )}

                    {messages.map((m) => (
                        <UserMessageView
                            onSave={onEdit}
                            onDelete={onDelete}
                            author={m.author}
                            message={m.message}
                            key={m.message.message_id}
                        />
                    ))}
                    <Paper elevation={1}>
                        <Stack spacing={2}>
                            <MDEditor
                                initialBodyMDValue={''}
                                onSave={onSave}
                                saveLabel={'Send Message'}
                            />
                        </Stack>
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
