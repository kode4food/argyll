import { useState, useEffect } from "react";

const useFocusWithin = (ref: React.RefObject<HTMLElement | null>): boolean => {
  const [focused, setFocused] = useState(false);

  useEffect(() => {
    const onFocusChange = () => {
      setFocused(!!ref.current?.contains(document.activeElement));
    };
    document.addEventListener("focusin", onFocusChange);
    document.addEventListener("focusout", onFocusChange);
    return () => {
      document.removeEventListener("focusin", onFocusChange);
      document.removeEventListener("focusout", onFocusChange);
    };
  }, [ref]);

  return focused;
};

export default useFocusWithin;
