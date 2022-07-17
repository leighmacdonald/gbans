import React, { useCallback, useEffect, useState, MouseEvent } from 'react';
import {
    apiCreateReportMessage,
    apiDeleteReportMessage,
    apiGetPersonConnections,
    apiGetPersonMessages,
    apiGetReportMessages,
    apiUpdateReportMessage,
    PermissionLevel,
    PersonConnection,
    PersonMessage,
    Report,
    ReportMessage,
    ReportMessagesResponse,
    UserProfile,
    IAPIBanRecord,
    BanReasons
} from '../api';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Box from '@mui/material/Box';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import Stack from '@mui/material/Stack';
import Avatar from '@mui/material/Avatar';
import { formatDistance, parseJSON } from 'date-fns';
import CardContent from '@mui/material/CardContent';
import CardHeader from '@mui/material/CardHeader';
import Card from '@mui/material/Card';
import IconButton from '@mui/material/IconButton';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { logErr } from '../util/errors';
import { renderMarkdown } from '../api/wiki';
import { MDEditor } from './MDEditor';
import { DataTable } from './DataTable';
import MenuItem from '@mui/material/MenuItem';
import Menu from '@mui/material/Menu';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import useTheme from '@mui/material/styles/useTheme';
import { RenderedMarkdownBox } from './RenderedMarkdownBox';
import Typography from '@mui/material/Typography';

interface TabPanelProps {
    children?: React.ReactNode;
    index: number;
    value: number;
}

function TabPanel(props: TabPanelProps) {
    const { children, value, index, ...other } = props;

    return (
        <div
            role="tabpanel"
            hidden={value !== index}
            id={`tabpanel-${index}`}
            aria-labelledby={`tab-${index}`}
            {...other}
        >
            {value === index && <Box sx={{ p: 0 }}>{children}</Box>}
        </div>
    );
}

export interface UserMessageViewProps {
    author: UserProfile;
    message: ReportMessage;
    onSave: (message: ReportMessage) => void;
    onDelete: (report_message_id: number) => void;
}

const UserMessageView = ({
    author,
    message,
    onSave,
    onDelete
}: UserMessageViewProps) => {
    const theme = useTheme();
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);
    const [editing, setEditing] = useState<boolean>(false);
    const [deleted, setDeleted] = useState<boolean>(false);
    const handleClick = (event: MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };
    const handleClose = () => {
        setAnchorEl(null);
    };
    if (deleted) {
        return <></>;
    }
    if (editing) {
        return (
            <Box component={Paper} padding={1}>
                <MDEditor
                    cancelEnabled
                    onCancel={() => {
                        setEditing(false);
                    }}
                    initialBodyMDValue={message.contents}
                    onSave={(body_md) => {
                        const newMsg = { ...message, contents: body_md };
                        onSave(newMsg);
                        message = newMsg;
                        setEditing(false);
                    }}
                />
            </Box>
        );
    } else {
        let d1 = formatDistance(parseJSON(message.created_on), new Date(), {
            addSuffix: true
        });
        if (message.updated_on != message.created_on) {
            d1 = `${d1} (edited: ${formatDistance(
                parseJSON(message.updated_on),
                new Date(),
                {
                    addSuffix: true
                }
            )})`;
        }
        return (
            <Card elevation={1}>
                <CardHeader
                    sx={{
                        backgroundColor: theme.palette.background.paper
                    }}
                    avatar={
                        <Avatar aria-label="Avatar" src={author.avatar}>
                            ?
                        </Avatar>
                    }
                    action={
                        <IconButton aria-label="Actions" onClick={handleClick}>
                            <MoreVertIcon />
                        </IconButton>
                    }
                    title={author.name}
                    subheader={d1}
                />
                <CardContent>
                    <RenderedMarkdownBox
                        bodyMd={renderMarkdown(message.contents)}
                    />
                </CardContent>
                <Menu
                    anchorEl={anchorEl}
                    id="message-menu"
                    open={open}
                    onClose={handleClose}
                    onClick={handleClose}
                    PaperProps={{
                        elevation: 0,
                        sx: {
                            overflow: 'visible',
                            filter: 'drop-shadow(0px 2px 8px rgba(0,0,0,0.32))',
                            mt: 1.5,
                            '& .MuiAvatar-root': {
                                width: 32,
                                height: 32,
                                ml: -0.5,
                                mr: 1
                            },
                            '&:before': {
                                content: '""',
                                display: 'block',
                                position: 'absolute',
                                top: 0,
                                right: 14,
                                width: 10,
                                height: 10,
                                bgcolor: 'background.paper',
                                transform: 'translateY(-50%) rotate(45deg)',
                                zIndex: 0
                            }
                        }
                    }}
                    transformOrigin={{ horizontal: 'right', vertical: 'top' }}
                    anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
                >
                    <MenuItem
                        onClick={() => {
                            setEditing(true);
                        }}
                    >
                        Edit
                    </MenuItem>
                    <MenuItem
                        onClick={() => {
                            onDelete(message.report_message_id);
                            setDeleted(true);
                        }}
                    >
                        Delete
                    </MenuItem>
                </Menu>
            </Card>
        );
    }
};

interface ReportComponentProps {
    report: Report;
    banHistory: IAPIBanRecord[];
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
            .then((r) => {
                setMessages(r || []);
            })
            .catch(logErr);
    }, [report.report_id]);

    const onSave = useCallback(
        (message: string) => {
            apiCreateReportMessage(report.report_id, message)
                .then((response) => {
                    setMessages([
                        ...messages,
                        { author: currentUser, message: response }
                    ]);
                })
                .catch(logErr);
        },
        [messages, report.report_id, currentUser]
    );

    const onEdit = useCallback(
        (message: ReportMessage) => {
            apiUpdateReportMessage(message.report_message_id, message.contents)
                .then(() => {
                    sendFlash('success', 'Updated message successfully');
                    loadMessages();
                })
                .catch(logErr);
        },
        [loadMessages, sendFlash]
    );

    const onDelete = useCallback(
        (report_message_id: number) => {
            apiDeleteReportMessage(report_message_id)
                .then(() => {
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
        apiGetPersonConnections(report.reported_id)
            .then((conns) => {
                setConnections(conns || []);
            })
            .catch(logErr);
    }, [report]);

    useEffect(() => {
        apiGetPersonMessages(report.reported_id)
            .then((msgs) => {
                setChatHistory(msgs || []);
            })
            .catch(logErr);
    }, [report]);

    return (
        <Grid container>
            <Grid item xs={12}>
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
                                    bodyMd={renderMarkdown(report.description)}
                                />
                            )}
                        </TabPanel>

                        <TabPanel value={value} index={1}>
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
                                        width: '150px'
                                    },
                                    {
                                        label: 'Message',
                                        tooltip: 'Message',
                                        sortKey: 'body',
                                        sortType: 'string',
                                        align: 'left'
                                    }
                                ]}
                                defaultSortColumn={'created_on'}
                                rowsPerPage={100}
                                rows={chatHistory}
                            />
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
                                        width: '150px'
                                    },
                                    {
                                        label: 'IP Address',
                                        tooltip: 'IP Address',
                                        sortKey: 'ipAddr',
                                        sortType: 'string',
                                        align: 'left'
                                    }
                                ]}
                                defaultSortColumn={'created_on'}
                                rowsPerPage={100}
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
                                        sortKey: 'author_id',
                                        sortType: 'string',
                                        align: 'left',
                                        width: '150px'
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
                                        label: 'Custom Reason',
                                        tooltip: 'Custom Reason',
                                        sortKey: 'reason_text',
                                        sortType: 'string',
                                        align: 'left'
                                    }
                                ]}
                                defaultSortColumn={'created_on'}
                                rowsPerPage={10}
                                rows={banHistory}
                            />
                        </TabPanel>
                    </Paper>

                    {messages.map((m) => (
                        <UserMessageView
                            onSave={onEdit}
                            onDelete={onDelete}
                            author={m.author}
                            message={m.message}
                            key={m.message.report_message_id}
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
