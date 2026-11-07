export interface QuestionTextParts {
  /** First paragraph: the actual question, set as the heading. */
  prompt: string
  /** Lines the DOCX importer placed between the question and its options. */
  context: string | null
  /** Context looks like source code (indented lines) and should keep monospace alignment. */
  contextIsCode: boolean
}

/**
 * The importer joins the question line and any following material (code,
 * numbered statements, matching tables) with "\n". The first line is the
 * question itself; the rest is shown as a separate block.
 */
export function splitQuestionText(text: string): QuestionTextParts {
  const normalized = text.replace(/\r\n?/g, '\n').replace(/^\n+|\n+$/g, '')
  const breakAt = normalized.indexOf('\n')
  if (breakAt === -1) return { prompt: normalized, context: null, contextIsCode: false }
  const prompt = normalized.slice(0, breakAt).trim()
  const context = normalized.slice(breakAt + 1).replace(/^\n+/, '')
  if (!prompt || !context.trim()) return { prompt: normalized.trim(), context: null, contextIsCode: false }
  const lines = context.split('\n')
  const contextIsCode =
    lines.some((line) => /^(\t| {2,})\S/.test(line)) ||
    lines.some((line) => /[{};]\s*$|^\s*(for|if|while|def|int|print|return|#include)\b/.test(line))
  return { prompt, context, contextIsCode }
}
