import React from 'react';
import { DataTable, RowsPerPage } from './DataTable';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';

export const NotificationList = () => {
    const { currentUser } = useCurrentUserCtx();
    return (
        <DataTable
            rows={currentUser.notifications || []}
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
