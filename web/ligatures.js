// Browser-safe programming ligatures via xterm.js character joiners (canvas renderer).
// Matches @xterm/addon-ligatures fallback set plus common Lua/markdown pairs.

export const FALLBACK_LIGATURES = [
  "..",
  "...",
  "<--",
  "<---",
  "<<-",
  "<-",
  "->",
  "->>",
  "-->",
  "--->",
  "<==",
  "<===",
  "<<=",
  "<=",
  "=>",
  "=>>",
  "==>",
  "===>",
  ">=",
  ">>=",
  "<->",
  "<-->",
  "<--->",
  "<---->",
  "<=>",
  "<==>",
  "<===>",
  "<====>",
  "-------->",
  "<~~",
  "<~",
  "~>",
  "~~>",
  "::",
  ":::",
  "==",
  "!=",
  "===",
  "!==",
  ":=",
  ":-",
  ":+",
  "<*",
  "<*>",
  "*>",
  "<|",
  "<|>",
  "|>",
  "+:",
  "-:",
  "=:",
  ":>",
  "++",
  "+++",
  "<!--",
  "<!---",
  "<***>",
].sort((a, b) => b.length - a.length);

export function ligatureRanges(text, patterns = FALLBACK_LIGATURES) {
  const ranges = [];
  for (let i = 0; i < text.length; i++) {
    for (const lig of patterns) {
      if (text.startsWith(lig, i)) {
        ranges.push([i, i + lig.length]);
        i += lig.length - 1;
        break;
      }
    }
  }
  return ranges;
}

/** @returns {() => void} dispose */
export function installLigatures(term, patterns = FALLBACK_LIGATURES) {
  if (!term?.registerCharacterJoiner) {
    return () => {};
  }
  const sorted = patterns.slice().sort((a, b) => b.length - a.length);
  const id = term.registerCharacterJoiner((text) => ligatureRanges(text, sorted));
  return () => {
    term.deregisterCharacterJoiner?.(id);
  };
}
