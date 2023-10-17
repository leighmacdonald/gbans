import NiceModal from '@ebay/nice-modal-react';
import { ContestEditor } from './ContestEditor';

export const ModalContestEditor = 'modal-contest-editor';

NiceModal.register(ModalContestEditor, ContestEditor);
