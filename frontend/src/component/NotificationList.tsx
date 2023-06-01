import React from 'react';
import { DataTable, RowsPerPage } from './DataTable';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';

export const NotificationList = () => {
    const { notifications } = useCurrentUserCtx();
    return (
        <DataTable
            rows={notifications}
            defaultSortColumn={'person_notification_id'}
            rowsPerPage={RowsPerPage.Fifty}
            columns={[
                {
                    label: 'Sev',
                    tooltip: 'Severity',
                    sortKey: 'severity',
                    sortType: 'number'
                },
                {
                    label: 'Message',
                    tooltip: 'Message',
                    sortKey: 'message'
                },
                {
                    label: 'Created',
                    tooltip: 'Created At',
                    sortKey: 'created_on',
                    sortType: 'date'
                }
            ]}
        />
    );
};
