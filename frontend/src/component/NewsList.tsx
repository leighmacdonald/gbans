import React from 'react';
import FolderIcon from '@mui/icons-material/Folder';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Stack from '@mui/material/Stack';
import { useTheme } from '@mui/material/styles';
import { NewsEntry } from '../api/news';
import { useNews } from '../hooks/useNews';

interface NewsListProps {
    setSelectedNewsEntry: (entry: NewsEntry) => void;
}

export const NewsList = ({ setSelectedNewsEntry }: NewsListProps) => {
    const theme = useTheme();
    const { data } = useNews();

    return (
        <Stack spacing={3} padding={3}>
            <List dense={true}>
                {data.map((entry) => {
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
