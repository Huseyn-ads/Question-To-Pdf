import { describe, expect, it } from 'vitest'
import { checkQuestion, cleanQuestion } from './questionRules'

const opt = (text: string, is_correct = false) => ({ text, is_correct })

describe('checkQuestion', () => {
  it('accepts a question with two unique options and one correct answer', () => {
    expect(checkQuestion('Столица Франции?', [opt('Париж', true), opt('Лион')]).valid).toBe(true)
  })

  it('requires text, two options and exactly one correct answer', () => {
    const result = checkQuestion('   ', [opt('A')])
    expect(result.valid).toBe(false)
    expect(result.question).toEqual(['text_required', 'options_min', 'correct_required'])
    expect(checkQuestion('Q', [opt('a', true), opt('b', true)]).question).toEqual(['correct_required'])
  })

  it('flags empty and case-insensitive duplicate options by index', () => {
    const result = checkQuestion('Q', [opt('Paris', true), opt(' paris '), opt('  '), opt('Lyon')])
    expect(result.options).toEqual({ 1: 'duplicate', 2: 'empty' })
    expect(result.valid).toBe(false)
  })

  it('treats Azerbaijani letters as distinct text', () => {
    expect(checkQuestion('Q', [opt('Şəki', true), opt('Səki')]).valid).toBe(true)
  })
})

describe('cleanQuestion', () => {
  it('trims text and options without dropping extra fields', () => {
    expect(cleanQuestion('  Q \n', [{ label: 'A', text: ' x ', is_correct: true }])).toEqual({
      text: 'Q',
      options: [{ label: 'A', text: 'x', is_correct: true }],
    })
  })
})
