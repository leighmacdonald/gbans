import NiceModal from '@ebay/nice-modal-react';
import { Confirm } from './Confirm';
import { ContestEditor } from './ContestEditor';
import { ContestEntryModal } from './ContestEntryModal';

export const ModalContestEditor = 'modal-contest-editor';
export const ModalContestEntry = 'modal-contest-entry';
export const ModalConfirm = 'modal-confirm';

NiceModal.register(ModalContestEditor, ContestEditor);
NiceModal.register(ModalContestEntry, ContestEntryModal);
NiceModal.register(ModalConfirm, Confirm);
