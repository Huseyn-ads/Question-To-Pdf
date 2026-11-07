import { describe, expect, it } from 'vitest'
import { loginPath, safeNext } from './session'

describe('safeNext', () => {
  it('keeps same-origin app paths with query and hash', () => {
    expect(safeNext('/quizzes/abc?x=1#top')).toBe('/quizzes/abc?x=1#top')
    expect(safeNext('/attempts/1?q=3')).toBe('/attempts/1?q=3')
  })

  it.each([
    'https://evil.example/',
    '//evil.example/path',
    '/\\evil.example',
    'javascript:alert(1)',
    'quizzes',
    '/v1/auth/me',
    '/v1/auth/google/start',
    '/login?next=/x',
    '/register',
    '/auth/complete',
    '/check-email',
    '/x\n/y',
    '',
    null,
  ])('rejects %s', (value) => {
    expect(safeNext(value)).toBe('/')
  })

  it('does not reject app paths that merely start like auth pages', () => {
    expect(safeNext('/loginx')).toBe('/loginx')
  })
})

describe('loginPath', () => {
  it('encodes the return path and drops unsafe or trivial ones', () => {
    expect(loginPath('/banks/1?tab=q')).toBe('/login?next=%2Fbanks%2F1%3Ftab%3Dq')
    expect(loginPath('/')).toBe('/login')
    expect(loginPath('//evil')).toBe('/login')
  })
})
