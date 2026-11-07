/** Option marker by display order: A, B, C… (matches DOCX labels students know). */
export const optionLetter = (index: number) => (index < 26 ? String.fromCharCode(65 + index) : String(index + 1))
