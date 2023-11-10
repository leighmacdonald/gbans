import React, { useCallback, useEffect, useState, JSX } from 'react';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import ReportIcon from '@mui/icons-material/Report';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Tab from '@mui/material/Tab';
import Tabs from '@mui/material/Tabs';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import {
    apiCreateReportMessage,
    apiDeleteReportMessage,
    apiGetMessages,
    apiGetReportMessages,
    apiUpdateReportMessage,
    PermissionLevel,
    PersonMessage,
    Report,
    ReportMessagesResponse,
    UserMessage
} from '../api';
import { renderMarkdown } from '../api/wiki';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { BanHistoryTable } from './BanHistoryTable';
import { ConnectionHistoryTable } from './ConnectionHistoryTable';
import { ContainerWithHeader } from './ContainerWithHeader';
import { MDEditor } from './MDEditor';
import { PersonMessageTable } from './PersonMessageTable';
import { PlayerMessageContext } from './PlayerMessageContext';
import { RenderedMarkdownBox } from './RenderedMarkdownBox';
import { SourceBansList } from './SourceBansList';
import { TabPanel } from './TabPanel';
import { UserMessageView } from './UserMessageView';

interface ReportComponentProps {
    report: Report;
}

export const ReportComponent = ({
    report
}: ReportComponentProps): JSX.Element => {
    const theme = useTheme();
    const [messages, setMessages] = useState<ReportMessagesResponse[]>([]);
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
                setMessages(response || []);
            })
            .catch(logErr);
    }, [report.report_id]);

    const onSave = useCallback(
        (message: string, onSuccess?: () => void) => {
            apiCreateReportMessage(report.report_id, message)
                .then((response) => {
                    setMessages([
                        ...messages,
                        { author: currentUser, message: response }
                    ]);
                    onSuccess && onSuccess();
                })
                .catch((e) => {
                    sendFlash('error', 'Failed to save report message');
                    logErr(e);
                });
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
                .then(() => {
                    sendFlash('success', 'Deleted message successfully');
                    loadMessages();
                })
                .catch((e) => {
                    sendFlash('error', 'Failed to delete message');
                    logErr(e);
                });
        },
        [loadMessages, sendFlash]
    );

    useEffect(() => {
        loadMessages();
    }, [loadMessages, report]);

    useEffect(() => {
        apiGetMessages({ steam_id: report.target_id, order_by: 'created_on' })
            .then((response) => {
                setChatHistory(response.messages);
            })
            .catch(logErr);
    }, [report]);

    return (
        <Grid container>
            <Grid xs={12}>
                <Stack spacing={2}>
                    <ContainerWithHeader
                        title={'Report Overview'}
                        iconLeft={<ReportIcon />}
                    >
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
                                    <Tab label={`Connection History`} />
                                )}
                                {currentUser.permission_level >=
                                    PermissionLevel.Moderator && (
                                    <Tab label={`Ban History`} />
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
                            <ConnectionHistoryTable
                                steam_id={report.target_id}
                            />
                        </TabPanel>
                        <TabPanel value={value} index={3}>
                            <BanHistoryTable steam_id={report.target_id} />
                        </TabPanel>
                    </ContainerWithHeader>
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
