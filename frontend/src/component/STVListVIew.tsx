import React from 'react';
import { apiGetDemos, DemoFile } from '../api';
import { ContainerWithHeader } from './ContainerWithHeader';
import { useCallback, useEffect, useState } from 'react';
import { DataTable, RowsPerPage } from './DataTable';
import { humanFileSize, renderDateTime } from '../util/text';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';

export interface STVListVIewProps {
    demos: DemoFile[];
}

export const STVListVIew = () => {
    const [demos, setDemos] = useState<DemoFile[]>([]);
    const [steamId] = useState('');
    const [mapName] = useState('');
    const [serverId] = useState(0);
    const [isLoading, setIsLoading] = useState(false);

    const reload = useCallback(() => {
        setIsLoading(true);
        apiGetDemos({ steamId, mapName, serverId })
            .then((response) => {
                setDemos(response.result ?? []);
            })
            .finally(() => {
                setIsLoading(false);
            });
    }, [mapName, serverId, steamId]);

    useEffect(() => {
        reload();
    }, [reload]);

    return (
        <ContainerWithHeader title={'SourceTV Recordings'}>
            <DataTable
                isLoading={isLoading}
                columns={[
                    {
                        tooltip: 'Server',
                        label: 'Server',
                        sortKey: 'server_name_short',
                        align: 'left'
                    },
                    {
                        tooltip: 'Demo Name',
                        label: 'Demo Name',
                        sortKey: 'title',
                        align: 'left'
                    },
                    {
                        tooltip: 'Size',
                        label: 'Size',
                        sortKey: 'size',
                        align: 'right',
                        renderer: (obj) => {
                            return humanFileSize(obj.size);
                        }
                    },
                    {
                        tooltip: 'Download',
                        label: 'DL',
                        virtual: true,
                        align: 'center',
                        virtualKey: 'Download',
                        renderer: (row) => {
                            return (
                                <IconButton
                                    component={Link}
                                    href={`/demos/${row.demo_id}`}
                                >
                                    <FileDownloadIcon />
                                </IconButton>
                            );
                        }
                    },
                    {
                        tooltip: 'Created On',
                        label: 'Created On',
                        sortKey: 'created_on',
                        align: 'right',
                        renderer: (row) => {
                            return renderDateTime(row.created_on);
                        }
                    }
                ]}
                defaultSortColumn={'created_on'}
                rowsPerPage={RowsPerPage.Fifty}
                rows={demos}
            />
        </ContainerWithHeader>
    );
};
