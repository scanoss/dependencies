// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2022 SCANOSS.COM
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 2 of the License, or
 * (at your option) any later version.
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

// Handle all interaction with the mines table

package models

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type MineModel struct {
	ctx  context.Context
	s    *zap.SugaredLogger
	conn *sqlx.Conn
}

type Mine struct {
	ID       int32  `db:"id"`
	Name     string `db:"mine_name"`
	PurlType string `db:"purl_type"`
}

// NewMineModel creates a new instance of the 'Mine' Model.
func NewMineModel(ctx context.Context, s *zap.SugaredLogger, conn *sqlx.Conn) *MineModel {
	return &MineModel{ctx: ctx, s: s, conn: conn}
}

// GetMineIdsByPurlType retrieves a list of the Purl Type IDs associated with the given Purl Type (string).
func (m *MineModel) GetMineIdsByPurlType(purlType string) ([]int32, error) {
	if len(purlType) == 0 {
		m.s.Error("Please specify a Purl Type to query")
		return nil, errors.New("please specify a Purl Type to query")
	}
	var mines []Mine
	err := m.conn.SelectContext(m.ctx, &mines,
		"SELECT id,mine_name,purl_type FROM mines WHERE purl_type = $1", purlType,
	)
	if err != nil {
		m.s.Errorf("Error: Failed to query mines table for %v: %v", purlType, err)
		return nil, fmt.Errorf("failed to query the mines table: %v", err)
	}
	if len(mines) > 0 {
		var mineIds []int32
		for _, mine := range mines {
			mineIds = append(mineIds, mine.ID)
		}
		return mineIds, nil
	}
	m.s.Error("No entries found in the mines table.")
	return nil, errors.New("no entry in mines table")
}
