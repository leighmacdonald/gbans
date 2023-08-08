import React from 'react';
import { App } from './App';
import { createRoot } from 'react-dom/client';
import { PaletteMode } from '@mui/material';

// extend window with our own items that we inject
declare global {
    interface Window {
        gbans: {
            site_name: string;
            discord_client_id: string;
            discord_link_id: string;
        };
    }
}

window.gbans = window.gbans || [];

const container = document.getElementById('root');
if (container) {
    createRoot(container).render(
        <App
            initialTheme={
                (localStorage.getItem('theme') as PaletteMode) || 'light'
            }
        />
    );
}
