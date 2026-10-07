import { describe, expect, it } from 'vitest'
import * as XLSX from 'xlsx'

describe('Patched SheetJS export compatibility', () => {
  it('writes the same usage workbook format with Unicode and appended rows', () => {
    const sheet = XLSX.utils.aoa_to_sheet([['用户', 'Model', 'Tokens']])
    XLSX.utils.sheet_add_aoa(sheet, [['测试用户', 'model-name', 123]], { origin: -1 })
    const book = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(book, sheet, 'Usage')
    const bytes = XLSX.write(book, { bookType: 'xlsx', type: 'array' })
    const loaded = XLSX.read(bytes, { type: 'array' })
    expect(XLSX.utils.sheet_to_json(loaded.Sheets.Usage!, { header: 1 })).toEqual([
      ['用户', 'Model', 'Tokens'], ['测试用户', 'model-name', 123],
    ])
  })
})
