// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026 Siyovush Hamidov and The Hadaf Contributors

package filters

import (
	"fmt"
	"time"
)

type NeedsFilter struct {
	CategoryID    *int       `form:"category_id"`
	Name          string     `form:"name"`
	Unit          string     `form:"unit"`
	RequiredQty   float64    `form:"required_qty"`
	ReceivedQty   float64    `form:"received_qty"`
	Urgency       string     `form:"urgency"`
	IsDone        *bool      `form:"is_done"`
	CreatedAtFrom *time.Time `form:"created_at_from" time_format:"2006-01-02"`
	CreatedAtTo   *time.Time `form:"created_at_to" time_format:"2006-01-02"`
	IsDeleted     bool       `form:"is_deleted"`
	OrderBy       string     `form:"order_by"`
}

func GetNeedsByInstitution(filter NeedsFilter, instituteId int) (string, []interface{}) {
	filterQuery := ""
	var args []interface{}

	// Parameter index to avoid confusion with argument sequence.
	idx := 1

	// is_deleted is always the first parameter.
	filterQuery += fmt.Sprintf(" WHERE n.is_deleted = $%d", idx)
	args = append(args, filter.IsDeleted)
	idx++

	filterQuery += fmt.Sprintf(" AND n.institution_id = $%d", idx)
	args = append(args, instituteId)
	idx++

	if filter.IsDone != nil {
		if *filter.IsDone {
			filterQuery += " AND n.received_qty >= n.required_qty"
		} else {
			filterQuery += " AND n.received_qty < n.required_qty"
		}
	}

	if filter.Name != "" {
		filterQuery += fmt.Sprintf(" AND n.name ILIKE $%d", idx)
		args = append(args, "%"+filter.Name+"%")
		idx++
	}

	if filter.CategoryID != nil {
		filterQuery += fmt.Sprintf(" AND n.category_id = $%d", idx)
		args = append(args, filter.CategoryID)
		idx++
	}

	if filter.Unit != "" {
		filterQuery += fmt.Sprintf(" AND n.unit ILIKE $%d", idx)
		args = append(args, filter.Unit)
		idx++
	}

	if filter.RequiredQty != 0 {
		filterQuery += fmt.Sprintf(" AND n.required_qty >= $%d", idx)
		args = append(args, filter.RequiredQty)
		idx++
	}

	if filter.ReceivedQty != 0 {
		filterQuery += fmt.Sprintf(" AND n.received_qty >= $%d", idx)
		args = append(args, filter.ReceivedQty)
		idx++
	}

	if filter.Urgency != "" {
		filterQuery += fmt.Sprintf(" AND n.urgency ILIKE $%d", idx)
		args = append(args, "%"+filter.Urgency+"%")
		idx++
	}

	if filter.CreatedAtFrom != nil {
		filterQuery += fmt.Sprintf(" AND n.created_at >= $%d", idx)
		args = append(args, *filter.CreatedAtFrom)
		idx++
	}
	if filter.CreatedAtTo != nil {
		// Add 24 hours to include the entire end of the day.
		to := filter.CreatedAtTo.Add(24 * time.Hour)
		filterQuery += fmt.Sprintf(" AND n.created_at < $%d", idx)
		args = append(args, to)
		idx++
	}

	// ORDER BY is strictly whitelisted.
	switch filter.OrderBy {
	case "date_asc":
		filterQuery += " ORDER BY n.created_at ASC"
	case "urgency":
		filterQuery += " ORDER BY n.urgency DESC, n.created_at DESC"
	default:
		filterQuery += " ORDER BY n.created_at DESC"
	}

	return filterQuery, args
}
