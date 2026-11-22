import { useEffect, useState } from "react";

export const MS_PER_SECOND = 1000;
export const COMPLETE_PERCENT = 100;

export const useWorkTimer = (start: number, end: number) => {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const interval = window.setInterval(
      () => setNow(Date.now()),
      MS_PER_SECOND
    );
    return () => window.clearInterval(interval);
  }, []);

  const duration = Math.max(1, end - start);
  const elapsed = Math.max(0, Math.min(duration, now - start));
  const remaining = Math.max(0, end - now);
  const value = Math.round((elapsed / duration) * COMPLETE_PERCENT);

  return { value, remaining };
};
