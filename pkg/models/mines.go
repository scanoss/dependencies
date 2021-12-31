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
	"errors"
	"github.com/jmoiron/sqlx"
	"log"
)

type mineModel struct {
	db *sqlx.DB
}

type Mine struct {
	Id       int    `db:"id"`
	Name     string `db:"name"`
	PurlType string `db:"purl_type"`
}

var minesCache map[string]Mine // TODO how long should this cache live?

func NewMineModel(db *sqlx.DB) *mineModel {
	return &mineModel{db: db}
}

func (m *mineModel) ResetMineCache() {
	minesCache = nil
}

func (m *mineModel) getMines() error {
	// Mines table already cached
	if minesCache != nil || len(minesCache) > 0 {
		return nil
	}
	log.Printf("Building mine cache...")
	minesCache = make(map[string]Mine)
	mine := Mine{}
	rows, err := m.db.Queryx("SELECT id,name,purl_type FROM mines")
	if err != nil {
		log.Printf("Error: Failed to query mines table: %v", err)
		return errors.New("failed to query the mines table")
	}
	for rows.Next() {
		err := rows.StructScan(&mine)
		if err != nil {
			log.Printf("Failed to parse row: %v", err)
			return errors.New("failed to parse mines row data")
		}
		minesCache[mine.PurlType] = Mine{Id: mine.Id, Name: mine.Name, PurlType: mine.PurlType}
	}
	return nil
}

func (m *mineModel) GetMineIdByPurlType(purlType string) (int, error) {
	if len(purlType) == 0 {
		log.Printf("Please specify a Purl Type to query")
		return -1, errors.New("please specify a Purl Type to query")
	}
	if m.getMines() != nil {
		log.Printf("Failed to build mines table cache")
		return -1, errors.New("failed to build mines table cache")
	}
	mine, ok := minesCache[purlType]
	if ok {
		log.Printf("Mine details for %s: %v", purlType, mine)
		return mine.Id, nil
	}
	return -1, errors.New("no entry in mines cache")
}
