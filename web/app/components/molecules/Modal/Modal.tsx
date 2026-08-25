import React from "react";
import styles from "./Modal.module.css";

export interface ModalProps {
  children: React.ReactNode;
  footer?: React.ReactNode;
  height?: number;
  isOpen: boolean;
  onClose: () => void;
  title: React.ReactNode;
  width?: number;
}

const Modal: React.FC<ModalProps> = ({
  children,
  footer,
  height,
  isOpen,
  onClose,
  title,
  width,
}) => {
  const dialogRef = React.useRef<HTMLDialogElement>(null);

  React.useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (isOpen && !dialog.open) dialog.showModal();
    if (!isOpen && dialog.open) dialog.close();
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <dialog
      ref={dialogRef}
      className={styles.dialog}
      data-ui-overlay="modal"
      onClose={onClose}
      onClick={(e) => {
        if (e.target === dialogRef.current) onClose();
      }}
      style={{
        ...(width && { width: `${width}px` }),
        ...(height && { height: `${height}px` }),
      }}
    >
      <div className={styles.header}>
        <h2 className={styles.title}>{title}</h2>
      </div>
      <div className={styles.body}>{children}</div>
      {footer && <div className={styles.footer}>{footer}</div>}
    </dialog>
  );
};

export default Modal;
