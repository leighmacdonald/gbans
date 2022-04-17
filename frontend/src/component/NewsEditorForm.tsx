import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import React, { useState } from 'react';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tabs from '@mui/material/Tabs';
import Tab from '@mui/material/Tab';
import { marked } from 'marked';

interface TabPanelProps {
    children?: React.ReactNode;
    index: number;
    value: number;
}

const TabPanel = (props: TabPanelProps) => {
    const { children, value, index, ...other } = props;

    return (
        <div
            role="tabpanel"
            hidden={value !== index}
            id={`tabpanel-${index}`}
            aria-labelledby={`ab-${index}`}
            {...other}
        >
            {value === index && (
                <Box sx={{ p: 3 }}>
                    <Typography>{children}</Typography>
                </Box>
            )}
        </div>
    );
};

export const NewsEditorForm = () => {
    const [headline, setHeadline] = useState<string>('');
    const [body, setBody] = useState<string>('');
    const [bodyHTML, setBodyHTML] = useState<string>('');
    const [value, setValue] = React.useState(0);

    const handleChange = (_: React.SyntheticEvent, newValue: number) => {
        setValue(newValue);
    };

    return (
        <Stack spacing={3} padding={3}>
            <Box color={'primary'}>
                <Typography variant={'h4'}>Create News Entry</Typography>
            </Box>
            <TextField
                id="headline"
                label="Headline"
                fullWidth
                value={headline}
                onChange={(v) => {
                    setHeadline(v.target.value);
                }}
            />
            <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                <Tabs
                    value={value}
                    onChange={handleChange}
                    aria-label="basic tabs example"
                >
                    <Tab label="Edit" />
                    <Tab label="Preview" />
                </Tabs>
            </Box>
            <TabPanel value={value} index={0}>
                <TextField
                    id="body"
                    label="Body (Markdown)"
                    fullWidth
                    multiline
                    minRows={15}
                    value={body}
                    onChange={(v) => {
                        setBody(v.target.value);
                        setBodyHTML(marked(v.target.value));
                    }}
                />
            </TabPanel>
            <TabPanel value={value} index={1}>
                <div dangerouslySetInnerHTML={{ __html: bodyHTML }} />
            </TabPanel>
        </Stack>
    );
};
