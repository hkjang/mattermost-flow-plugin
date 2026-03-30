import React, {useCallback, useEffect, useRef} from 'react';

type ConfirmDialogProps = {
    title: string;
    message: string;
    confirmLabel?: string;
    cancelLabel?: string;
    danger?: boolean;
    onConfirm: () => void;
    onCancel: () => void;
};

export function ConfirmDialog({
    title,
    message,
    confirmLabel = 'Confirm',
    cancelLabel = 'Cancel',
    danger = false,
    onConfirm,
    onCancel,
}: ConfirmDialogProps) {
    const cancelRef = useRef<HTMLButtonElement>(null);

    useEffect(() => {
        cancelRef.current?.focus();
    }, []);

    const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
        if (event.key === 'Escape') {
            event.stopPropagation();
            onCancel();
        }
    }, [onCancel]);

    const handleBackdropClick = useCallback((event: React.MouseEvent) => {
        if (event.target === event.currentTarget) {
            onCancel();
        }
    }, [onCancel]);

    return (
        <div // eslint-disable-line jsx-a11y/no-static-element-interactions
            className='flow-dialog-backdrop'
            onClick={handleBackdropClick}
            onKeyDown={handleKeyDown}
        >
            <div
                className='flow-dialog'
                role='dialog'
                aria-modal='true'
                aria-labelledby='flow-dialog-title'
            >
                <h3 id='flow-dialog-title'>{title}</h3>
                <p>{message}</p>
                <div className='flow-dialog__actions'>
                    <button
                        ref={cancelRef}
                        className='flow-button'
                        onClick={onCancel}
                        type='button'
                    >
                        {cancelLabel}
                    </button>
                    <button
                        className={`flow-button ${danger ? 'flow-button--danger' : 'flow-button--primary'}`}
                        onClick={onConfirm}
                        type='button'
                    >
                        {confirmLabel}
                    </button>
                </div>
            </div>
        </div>
    );
}
