//! Tables, laid out as Go's `text/tabwriter` with padding 2 and spaces.

use super::cells::clean_cell;

/// One table column over rows of type `T`.
pub struct Column<T> {
    pub header: &'static str,
    pub cell: fn(&T) -> String,
}

pub fn col<T>(header: &'static str, cell: fn(&T) -> String) -> Column<T> {
    Column { header, cell }
}

/// Go's tabwriter (minwidth 0, tabwidth 0, padding 2, space padding, no
/// flags) for lines whose cells are already split: every cell but the
/// last is padded to its column's widest cell plus two; the last is
/// written as it is. Widths count runes, as Go does.
pub fn tabwrite(lines: &[Vec<String>]) -> String {
    let mut out = String::new();
    let mut start = 0;
    while start < lines.len() {
        // A block is a run of lines with the same number of cells.
        let cells = lines[start].len();
        let end = start
            + lines[start..]
                .iter()
                .take_while(|l| l.len() == cells)
                .count();
        let block = &lines[start..end];
        let widths: Vec<usize> = (0..cells.saturating_sub(1))
            .map(|i| {
                block
                    .iter()
                    .map(|l| l[i].chars().count())
                    .max()
                    .unwrap_or(0)
                    + 2
            })
            .collect();
        for line in block {
            for (i, cell) in line.iter().enumerate() {
                out.push_str(cell);
                if let Some(width) = widths.get(i) {
                    out.push_str(&" ".repeat(width - cell.chars().count()));
                }
            }
            out.push('\n');
        }
        start = end;
    }
    out
}

/// The Go CLI's `printTable`: a header line, then one line per row, each
/// cell cleaned of control characters.
pub fn render<T>(rows: &[T], columns: &[Column<T>]) -> String {
    let mut lines = vec![
        columns
            .iter()
            .map(|c| c.header.to_string())
            .collect::<Vec<_>>(),
    ];
    for row in rows {
        lines.push(columns.iter().map(|c| clean_cell(&(c.cell)(row))).collect());
    }
    tabwrite(&lines)
}

#[cfg(test)]
mod tests {
    use super::tabwrite;

    // Fault: columns did not line up as in the Go CLI (wrong padding, byte
    // widths instead of rune widths, or padding after the last column).
    #[test]
    fn layout_matches_go_tabwriter() {
        let lines = vec![
            vec!["ID".to_string(), "NAME".to_string(), "STATUS".to_string()],
            vec![
                "nas01".to_string(),
                "Büro".to_string(),
                "Online".to_string(),
            ],
            vec!["a".to_string(), "b".to_string(), "-".to_string()],
        ];
        assert_eq!(
            tabwrite(&lines),
            "ID     NAME  STATUS\nnas01  Büro  Online\na      b     -\n"
        );
    }
}
