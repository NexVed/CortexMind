import { Component, JSX, Show } from 'solid-js';
import { X } from 'lucide-solid';
import './Modal.css';

interface Props {
  open: boolean;
  title: string;
  onClose: () => void;
  children: JSX.Element;
}

// Lightweight modal used by the create flows across pages.
export const Modal: Component<Props> = (props) => {
  return (
    <Show when={props.open}>
      <div class="cx-modal-overlay" onClick={props.onClose}>
        <div class="cx-modal" onClick={(e) => e.stopPropagation()}>
          <div class="cx-modal-header">
            <h3>{props.title}</h3>
            <button class="cx-modal-close" onClick={props.onClose} aria-label="Close">
              <X size={18} />
            </button>
          </div>
          <div class="cx-modal-body">{props.children}</div>
        </div>
      </div>
    </Show>
  );
};
