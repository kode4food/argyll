import { useRef, useEffect, useLayoutEffect } from "react";

const useClickOutside = (
  ref: React.RefObject<HTMLElement | null>,
  onClickOutside: () => void,
  active: boolean
): void => {
  const callbackRef = useRef(onClickOutside);
  useLayoutEffect(() => {
    callbackRef.current = onClickOutside;
  });

  useEffect(() => {
    if (!active) return;
    const handler = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) {
        callbackRef.current();
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [active, ref]);
};

export default useClickOutside;
