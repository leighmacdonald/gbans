import React, { useEffect, useState } from 'react';
import useTheme from '@mui/material/styles/useTheme';
import ListItem from '@mui/material/ListItem';
import List from '@mui/material/List';
import Stack from '@mui/material/Stack';
import { apiGetNewsAll, NewsEntry } from '../api/news';
import FolderIcon from '@mui/icons-material/Folder';
import { logErr } from '../util/errors';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';

interface NewsListProps {
    setSelectedNewsEntry: (entry: NewsEntry) => void;
}

export const NewsList = ({ setSelectedNewsEntry }: NewsListProps) => {
    const [news, setNews] = useState<NewsEntry[]>([]);
    const theme = useTheme();
    useEffect(() => {
        apiGetNewsAll()
            .then((r) => {
                setNews(r || []);
            })
            .catch(logErr);
    }, []);
    return (
        <Stack spacing={3} padding={3}>
            <List dense={true}>
                {news.map((entry) => {
                    return (
                        <ListItem
                            sx={[
                                {
                                    '&:hover': {
                                        cursor: 'pointer',
                                        backgroundColor:
                                            theme.palette.background.default
                                    }
                                }
                            ]}
                            key={entry.news_id}
                            onClick={() => {
                                setSelectedNewsEntry(entry);
                            }}
                        >
                            <ListItemIcon>
                                <FolderIcon />
                            </ListItemIcon>
                            <ListItemText primary={entry.title} />
                        </ListItem>
                    );
                })}
            </List>
        </Stack>
    );
};
