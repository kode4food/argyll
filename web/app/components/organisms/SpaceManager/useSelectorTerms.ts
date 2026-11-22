import React from "react";

export const useSelectorTerms = (
  terms: string[][],
  onTermsChange: (qbe: string[][]) => void
) => {
  const [focusTermIndex, setFocusTermIndex] = React.useState(-1);

  const handleAddTerm = () => {
    setFocusTermIndex(terms.length);
    onTermsChange([...terms, []]);
  };

  const handleTermChange = (index: number, tags: string[]) =>
    onTermsChange(terms.map((current, at) => (at === index ? tags : current)));

  const handleRemoveTerm = (index: number) =>
    onTermsChange(terms.filter((_, at) => at !== index));

  const shouldFocusTerm = (index: number) =>
    index === focusTermIndex || (terms.length === 1 && !terms[index].length);

  return {
    handleAddTerm,
    handleTermChange,
    handleRemoveTerm,
    shouldFocusTerm,
  };
};
