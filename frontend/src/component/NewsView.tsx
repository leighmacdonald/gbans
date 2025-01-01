import { useEffect } from 'react';
import { useState } from 'react';
import { Pagination } from '@mui/material';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { apiGetNewsLatest, NewsEntry } from '../api/news';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { logErr } from '../util/errors';
import { renderDate } from '../util/time.ts';
import { MarkDownRenderer } from './MarkdownRenderer';
import { SplitHeading } from './SplitHeading';

export interface NewsViewProps {
    itemsPerPage: number;
}

export const NewsView = ({ itemsPerPage }: NewsViewProps) => {
    const { sendFlash } = useUserFlashCtx();
    const [articles, setArticles] = useState<NewsEntry[]>([]);
    const [page, setPage] = useState<number>(0);

    useEffect(() => {
        const abortController = new AbortController();
        const fetchNews = async () => {
            try {
                const response = await apiGetNewsLatest(abortController);
                setArticles(response);
            } catch (error) {
                logErr(error);
            }
        };

        fetchNews().catch(logErr);

        return () => abortController.abort();
    }, [itemsPerPage, sendFlash]);

    return (
        <Stack spacing={3}>
            {(articles || [])?.slice(page * itemsPerPage, page * itemsPerPage + itemsPerPage).map((article) => {
                if (!article.created_on || !article.updated_on) {
                    return null;
                }
                return (
                    <Paper elevation={1} key={`news_` + article.news_id}>
                        <SplitHeading left={article.title} right={renderDate(article.created_on)} />
                        <MarkDownRenderer body_md={article.body_md} />
                    </Paper>
                );
            })}
            <Pagination
                count={articles ? Math.ceil(articles.length / itemsPerPage) : 0}
                defaultValue={1}
                onChange={(_, newPage) => {
                    setPage(newPage - 1);
                }}
            ></Pagination>
        </Stack>
    );
};
