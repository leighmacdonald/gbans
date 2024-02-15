import { Component, ErrorInfo, ReactNode } from 'react';
import Typography from '@mui/material/Typography';
import { logErr } from '../util/errors';

interface BoundaryState {
    hasError: boolean;
}

interface BoundaryProps {
    children: ReactNode;
}

export class ErrorBoundary extends Component<BoundaryProps, BoundaryState> {
    constructor(props: BoundaryProps) {
        super(props);
        this.state = { hasError: false };
    }

    static getDerivedStateFromError() {
        return { hasError: true };
    }

    componentDidCatch(error: Error, errorInfo: ErrorInfo) {
        // TODO record somewhere, sentry?
        logErr(error);
        logErr(errorInfo);
    }

    render(): ReactNode {
        if (this.state.hasError) {
            return (
                <Typography
                    marginTop={3}
                    variant={'h2'}
                    color={'error'}
                    textAlign={'center'}
                >
                    🤯 🤯 🤯 Something went wrong 🤯 🤯 🤯
                </Typography>
            );
        }
        return this.props.children;
    }
}
