import { describe, expect, it } from 'vitest'
import { splitQuestionText } from './questionText'

describe('splitQuestionText', () => {
  it('keeps single-line questions intact', () => {
    expect(splitQuestionText('Столица Франции?')).toEqual({ prompt: 'Столица Франции?', context: null, contextIsCode: false })
  })

  it('separates code context and detects indentation', () => {
    const parts = splitQuestionText("Что выведет фрагмент?\n\nfor i in range(3):\n    print(i)")
    expect(parts.prompt).toBe('Что выведет фрагмент?')
    expect(parts.context).toBe('for i in range(3):\n    print(i)')
    expect(parts.contextIsCode).toBe(true)
  })

  it('treats numbered statements as plain context', () => {
    const parts = splitQuestionText('Hansı ifadələr doğrudur?\n1. Birinci ifadə\n2. İkinci ifadə')
    expect(parts.contextIsCode).toBe(false)
    expect(parts.context).toBe('1. Birinci ifadə\n2. İkinci ifadə')
  })
})
