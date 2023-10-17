import NiceModal from '@ebay/nice-modal-react';
import { ContestEditor } from './ContestEditor';
import { Confirm } from './Confirm';

export const ModalContestEditor = 'modal-contest-editor';
export const ModalConfirm = 'modal-confirm';

NiceModal.register(ModalContestEditor, ContestEditor);
NiceModal.register(ModalConfirm, Confirm);
