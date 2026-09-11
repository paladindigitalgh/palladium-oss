/** A CSV column: `key` looks up the value on each row, `label` is the header cell. */
export interface CsvColumn {
  key: string
  label: string
}

/**
 * Quotes a single CSV field per RFC 4180: wrapped in double quotes only
 * when it contains a comma, a double quote, or a newline, with any
 * embedded double quote doubled up. A field needing no quoting is
 * returned unchanged, not just for a smaller output -- Excel keeps a
 * lone unquoted `""` result of any earlier quoting bug hard to spot at a
 * glance, whereas an unquoted field is at a glance still exactly the
 * value it holds.
 */
function quoteField(value: string): string {
  if (/[",\n]/.test(value)) {
    return `"${value.replace(/"/g, '""')}"`
  }
  return value
}

/**
 * Renders `rows` as a CSV string: a header row from `columns[].label`,
 * then one row per entry, reading `columns[].key` out of each — the
 * export format behind Explorer's "Export CSV" button
 * (docs/09-WORKSPACE-SPECIFICATIONS.md §15's "Export results to CSV"
 * Primary Action). Ends with a trailing newline, the conventional CSV
 * file shape most spreadsheet tools expect.
 */
export function rowsToCsv(columns: CsvColumn[], rows: Record<string, string>[]): string {
  const lines = [columns.map((c) => quoteField(c.label)).join(',')]
  for (const row of rows) {
    lines.push(columns.map((c) => quoteField(row[c.key] ?? '')).join(','))
  }
  return lines.join('\n') + '\n'
}

/**
 * Triggers a browser download of `content` as a file named `filename`.
 * The standard client-side download pattern -- a Blob, a temporary
 * `<a download>`, `URL.createObjectURL`/`revokeObjectURL` -- needed
 * because Explorer's reports are fetched once as JSON and turned into
 * CSV entirely client-side (see reportRepository.ts's own doc comment);
 * there is no backend CSV endpoint to link to instead.
 */
export function downloadCsv(filename: string, content: string): void {
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
