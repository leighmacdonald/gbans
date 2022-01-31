import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import Paper from '@mui/material/Paper';
import ListSubheader from '@mui/material/ListSubheader';
import Stack from '@mui/material/Stack';
import { ReportComponent } from '../component/ReportComponent';
import { useParams } from 'react-router-dom';
import { apiGetReport, ReportWithAuthor } from '../api';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import SendIcon from '@mui/icons-material/Send';
import Button from '@mui/material/Button';
import Avatar from '@mui/material/Avatar';
import ListItemAvatar from '@mui/material/ListItemAvatar';
import ListItemText from '@mui/material/ListItemText';

export const ReportViewPage = (): JSX.Element => {
    const { report_id } = useParams();
    const id = parseInt(report_id || '');
    const [report, setReport] = useState<ReportWithAuthor>();
    const [modAction, setModAction] = React.useState('');

    const handleChange = (event: SelectChangeEvent) => {
        setModAction(event.target.value as string);
    };

    useEffect(() => {
        const loadReport = async () => {
            const resp = await apiGetReport(id);
            setReport(resp);
        };
        loadReport();
    }, [report_id, setReport, id]);

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={9}>
                {report && <ReportComponent report={report.report} />}
            </Grid>
            <Grid item xs={3}>
                <Stack spacing={2}>
                    <Paper elevation={1}>
                        <List
                            subheader={
                                <ListSubheader
                                    component="div"
                                    id="nested-list-subheader"
                                >
                                    Moderation Tools
                                </ListSubheader>
                            }
                        >
                            <ListItem>
                                <Stack sx={{ width: '100%' }} spacing={2}>
                                    <FormControl fullWidth>
                                        <InputLabel id="select-label">
                                            Action
                                        </InputLabel>
                                        <Select
                                            labelId="select-label"
                                            id="simple-select"
                                            value={modAction}
                                            label="Report State"
                                            onChange={handleChange}
                                        >
                                            <MenuItem value={0}>
                                                Opened
                                            </MenuItem>
                                            <MenuItem value={1}>
                                                Need More Info
                                            </MenuItem>
                                            <MenuItem value={2}>
                                                Closed
                                            </MenuItem>
                                            <MenuItem value={3}>
                                                Closed (Banned)
                                            </MenuItem>
                                        </Select>
                                    </FormControl>
                                    <Button
                                        fullWidth
                                        variant={'contained'}
                                        color={'primary'}
                                        endIcon={<SendIcon />}
                                    >
                                        Set State
                                    </Button>
                                </Stack>
                            </ListItem>
                        </List>
                    </Paper>

                    <Paper elevation={1} sx={{ width: '100%' }}>
                        <List
                            sx={{ width: '100%' }}
                            subheader={
                                <ListSubheader
                                    component="div"
                                    id="nested-list-subheader"
                                >
                                    Reporter
                                </ListSubheader>
                            }
                        >
                            <ListItem>
                                <ListItemAvatar>
                                    <Avatar src={report?.author.avatar}>
                                        <SendIcon />
                                    </Avatar>
                                </ListItemAvatar>
                                <ListItemText
                                    primary={report?.author.personaname}
                                    secondary={'Reports: 12'}
                                />
                            </ListItem>
                        </List>
                    </Paper>

                    <Paper elevation={1}>
                        <List
                            subheader={
                                <ListSubheader
                                    component="div"
                                    id="nested-list-subheader"
                                >
                                    Report History
                                </ListSubheader>
                            }
                        />
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
