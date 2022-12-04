import React from 'react';
import { App } from './App';
import { createRoot } from 'react-dom/client';
import { PaletteMode } from '@mui/material';

// extend window with our own items that we inject
declare global {
    interface Window {
        gbans: {
            siteName: string;
            discordClientId: string;
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
