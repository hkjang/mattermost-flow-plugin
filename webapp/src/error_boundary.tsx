import React, {Component} from 'react';
import type {ErrorInfo, ReactNode} from 'react';

type Props = {
    children: ReactNode;
};

type State = {
    hasError: boolean;
    errorMessage: string;
};

export class FlowErrorBoundary extends Component<Props, State> {
    constructor(props: Props) {
        super(props);
        this.state = {hasError: false, errorMessage: ''};
    }

    static getDerivedStateFromError(error: Error): State {
        return {hasError: true, errorMessage: error.message || 'An unexpected error occurred'};
    }

    componentDidCatch(error: Error, info: ErrorInfo) {
        // eslint-disable-next-line no-console
        console.error('[Flow Plugin] Uncaught error:', error, info.componentStack);
    }

    handleReload = () => {
        this.setState({hasError: false, errorMessage: ''});
    };

    render() {
        if (this.state.hasError) {
            return (
                <div className='flow-error-boundary'>
                    <div className='flow-error-boundary__icon'>!</div>
                    <h3>{'Something went wrong'}</h3>
                    <p>{this.state.errorMessage}</p>
                    <button
                        className='flow-button flow-button--primary'
                        onClick={this.handleReload}
                        type='button'
                    >
                        {'Reload plugin'}
                    </button>
                </div>
            );
        }
        return this.props.children;
    }
}
