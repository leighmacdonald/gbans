import React, { useEffect, useState } from 'react';
import {
    apiGetReport,
    apiGetReportMessages,
    Report,
    ReportMessage
} from '../api';
import { useParams } from 'react-router-dom';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import ListItem from '@mui/material/ListItem';
import TextField from '@mui/material/TextField';
import SendIcon from '@mui/icons-material/Send';
import Button from '@mui/material/Button';
import Box from '@mui/material/Box';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';

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
            id={`simple-tabpanel-${index}`}
            aria-labelledby={`simple-tab-${index}`}
            {...other}
        >
            {value === index && (
                <Box sx={{ p: 3 }}>
                    <Typography>{children}</Typography>
                </Box>
            )}
        </div>
    );
}

export const ReportComponent = (): JSX.Element => {
    const { report_id } = useParams();
    const id = parseInt(report_id || '');
    const [comment, setComment] = useState<string>('');
    const [report, setReport] = useState<Report>();
    const [messages, setMessages] = useState<ReportMessage[]>([]);
    const [value, setValue] = React.useState<number>(0);

    const handleChange = (_: React.SyntheticEvent, newValue: number) => {
        setValue(newValue);
    };

    useEffect(() => {
        const loadReport = async () => {
            const resp = await apiGetReport(id);
            setReport(resp);
        };
        loadReport();
        const loadMessages = async () => {
            const resp = await apiGetReportMessages(id);
            setMessages(resp);
        };
        loadMessages();
    }, [report_id, setReport, id]);

    return (
        <>
            <Grid container>
                <Grid item xs={12}>
                    <Typography variant={'h2'}>{report?.title}</Typography>
                    <Box sx={{ width: '100%' }}>
                        <Box
                            sx={{
                                borderBottom: 1,
                                borderColor: 'divider'
                            }}
                        >
                            <Tabs
                                value={value}
                                onChange={handleChange}
                                aria-label="Report detail tabs"
                            >
                                <Tab label="Description" />
                                <Tab label="Chat Logs" />
                                <Tab label="Connections" />
                            </Tabs>
                        </Box>

                        <TabPanel value={value} index={0}>
                            {report && (
                                <Typography variant={'body1'}>
                                    {report.description}
                                </Typography>
                            )}
                        </TabPanel>
                        <TabPanel value={value} index={1}>
                            Chat history
                        </TabPanel>
                        <TabPanel value={value} index={2}>
                            Connection history
                        </TabPanel>
                    </Box>
                    <Paper elevation={1}>
                        {messages &&
                            messages.map((msg) => {
                                return (
                                    <ListItem key={msg.report_message_id}>
                                        <Typography variant={'h2'}>
                                            {msg.author_id}
                                        </Typography>
                                        <Typography variant={'body1'}>
                                            {msg.contents}
                                        </Typography>
                                    </ListItem>
                                );
                            })}
                        <Paper elevation={1} sx={{ marginTop: '1rem' }}>
                            <TextField
                                label="Comment"
                                id="comment"
                                minRows={10}
                                variant={'filled'}
                                margin={'normal'}
                                multiline
                                fullWidth
                                value={comment}
                                onChange={(v) => {
                                    setComment(v.target.value);
                                }}
                            />
                            <Button
                                fullWidth
                                variant={'contained'}
                                color={'primary'}
                                endIcon={<SendIcon />}
                            >
                                Send Comment
                            </Button>
                        </Paper>
                    </Paper>
                </Grid>
            </Grid>
        </>
    );
};
