import { Component, ErrorInfo, ReactNode } from 'react';
import { logErr } from '../util/errors';
import { ErrorNotice } from './ErrorNotice.tsx';

interface BoundaryState {
    hasError: boolean;
    error?: Error;
}

interface BoundaryProps {
    children?: ReactNode;
    fallback?: ReactNode;
}

export class ErrorBoundary extends Component<BoundaryProps, BoundaryState> {
    constructor(props: BoundaryProps) {
        super(props);
        this.state = { hasError: false };
    }

    static getDerivedStateFromError(error: Error) {
        // Update state so the next render will show the fallback UI.
        return { hasError: true, error: error };
    }

    componentDidCatch(error: Error, info: ErrorInfo) {
        if (error) {
            logErr(error);
        }
        if (info.componentStack) {
            logErr(info.componentStack);
        }
    }

    render() {
        if (this.state.hasError) {
            if (this.props.fallback) {
                // You can render any custom fallback UI
                return this.props.fallback;
            }
            return <ErrorNotice {...this.props} error={this.state.error} />;
        }

        return this.props.children;
    }
}
