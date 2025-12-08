import { ReactNode } from 'react';
import Button from '@mui/material/Button';

type TabButtonProps<Tabs> = {
    label: string;
    tab: Tabs;
    onClick: (tab: Tabs) => void;
    currentTab: Tabs;
    icon: ReactNode;
};

export const TabButton = <Tabs,>({ currentTab, tab, label, onClick, icon }: TabButtonProps<Tabs>) => {
    return (
        <Button
            color={currentTab == tab ? 'secondary' : 'primary'}
            onClick={() => onClick(tab)}
            variant={'contained'}
            startIcon={icon}
            fullWidth
            title={label}
            style={{ justifyContent: 'flex-start' }}
        >
            {label}
        </Button>
    );
};
