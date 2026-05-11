package output

import "fmt"

type QueryResult struct {
	Columns            []string `json:"columns"`
	Rows               [][]any  `json:"rows"`
	Page               int      `json:"page"`
	PageSize           int      `json:"page_size"`
	ReturnedRows       int      `json:"returned_rows"`
	HasMore            bool     `json:"has_more"`
	NextPage           *int     `json:"next_page"`
	TotalRowsExact     *int     `json:"total_rows_exact"`
	RemainingRowsExact *int     `json:"remaining_rows_exact"`
	Truncated          bool     `json:"truncated"`
	TruncatedCells     int      `json:"truncated_cells"`
}

func FormatRows(columns []string, rows [][]any, page, pageSize, truncateAt int) QueryResult {
	res := QueryResult{
		Columns:  columns,
		Page:     page,
		PageSize: pageSize,
	}

	if len(rows) > pageSize {
		res.HasMore = true
		nextPage := page + 1
		res.NextPage = &nextPage
		rows = rows[:pageSize]
	}

	formatted := make([][]any, 0, len(rows))
	for _, row := range rows {
		formattedRow := make([]any, len(row))
		for i, value := range row {
			switch typed := value.(type) {
			case string:
				if len(typed) > truncateAt {
					formattedRow[i] = typed[:truncateAt] + "..."
					res.Truncated = true
					res.TruncatedCells++
					continue
				}
				formattedRow[i] = typed
			case []byte:
				formattedRow[i] = fmt.Sprintf("[BINARY %dB]", len(typed))
				res.Truncated = true
				res.TruncatedCells++
			default:
				formattedRow[i] = value
			}
		}
		formatted = append(formatted, formattedRow)
	}

	res.Rows = formatted
	res.ReturnedRows = len(formatted)
	return res
}
